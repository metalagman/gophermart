package syncer

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/pkg/accrual"
	"runtime"
	"sync"
	"time"
)

const (
	statusProcessed = "PROCESSED"
)

var ErrRetryableError = errors.New("retryable")

type Job func() error

type Service struct {
	mu     sync.RWMutex
	logger logger.Logger
	db     *sql.DB

	accrual *accrual.Service
	jobs    chan Job
	stopCh  chan struct{}

	fetchInterval time.Duration
}

func New(db *sql.DB, ac *accrual.Service) (*Service, error) {
	s := &Service{
		logger:        logger.Global().WithComponent("AccrualSync.Service"),
		fetchInterval: 5 * time.Second,
		jobs:          make(chan Job),
		stopCh:        make(chan struct{}),
		accrual:       ac,
		db:            db,
	}
	s.Start(runtime.GOMAXPROCS(0))

	return s, nil
}

func (s *Service) Start(numWorkers int) {
	const retryDelay = time.Second
	for i := 0; i < numWorkers; i++ {
		go func(workerID int, l logger.Logger, jobs chan Job, stop chan struct{}) {
			for {
				select {
				case <-stop:
					close(jobs)
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					id := uuid.New()
					ll := l.With().Int("worker_id", workerID).Str("job_id", id.String()).Logger()
					ll.Info().Msg("Running job")
					if err := job(); err != nil {
						ll.Error().Msg("Job failed")
						go func() {
							ll.Info().Msg("Retrying job")
							time.Sleep(retryDelay)
							jobs <- job
						}()
						continue
					}
					ll.Info().Msg("Job done")
				}
			}
		}(i, s.logger, s.jobs, s.stopCh)
	}

	go func(l logger.Logger, fetchInterval time.Duration) {
		t := time.NewTimer(fetchInterval)
		for {
			select {
			case <-s.stopCh:
				t.Stop()
				return
			case <-t.C:
				l.Info().Msg("Fetching statuses")
				//s.jobs <- s.fetchStatuses()
				t.Reset(fetchInterval)
			}
		}

	}(s.logger, s.fetchInterval)

	return
}

func (s *Service) Stop() {
	s.logger.Debug().Msg("Service shutdown")
	close(s.stopCh)
}

func (s *Service) Run(job Job) {
	s.jobs <- job
}

func (s *Service) FetchOrderDetails(id uuid.UUID) Job {
	timeout := time.Second * 30
	return func() error {
		l := s.logger.WithComponent("AccrualSync.Job.FetchStatus")
		l.Debug().Msg("Fetching status")

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		ctx = l.WithContext(ctx)

		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		})
		if err != nil {
			l.Error().Err(err).Msg("DB transaction begin")
			return err
		}
		var oldStatus, externalID string
		var userID uuid.UUID
		const sqlLock = `SELECT status, external_id, user_id FROM orders WHERE id=$1 FOR UPDATE`
		if err := tx.QueryRowContext(ctx, sqlLock, id).Scan(&oldStatus, externalID, userID); err != nil {
			l.Error().Err(err).Msg("DB lock error")
			_ = tx.Rollback()
			return err
		}

		now := time.Now()

		in := &accrual.GetOrderRequest{
			ExternalOrderID: externalID,
		}
		out := &accrual.GetOrderResponse{}

		if err := s.accrual.GetOrder(ctx, in, out); err != nil {
			l.Error().Err(err).Msg("Status fetch failed")
			_ = tx.Rollback()
			return err
		}

		const sqlUpdate = `UPDATE orders SET status=$1, accrual=$2 WHERE id=$3`
		_, err = tx.ExecContext(ctx, sqlUpdate, out.Status, out.Accrual, id)
		if err != nil {
			l.Error().Err(err).Msg("Status update failed")
			_ = tx.Rollback()
			return err
		}

		if oldStatus != out.Status && out.Status == statusProcessed && out.Accrual.Valid {
			const sqlTx = `INSERT INTO transactions (type_id, user_id, order_id, amount) VALUES ($1, $2, $3, $4)`
			_, err = tx.ExecContext(ctx, sqlTx, model.TransactionTypeReplenishment, userID, id, out.Accrual)
			if err != nil {
				l.Error().Err(err).Msg("TX insert failed")
				_ = tx.Rollback()
				return err
			}

			const sqlUpdateBalance = `UPDATE users SET balance=balance+$1 WHERE id=$2`
			_, err = tx.ExecContext(ctx, sqlUpdateBalance, out.Accrual, userID)
			if err != nil {
				l.Error().Err(err).Msg("Balance update failed")
				_ = tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			l.Error().Err(err).Msg("TX commit failed")
			return err
		}

		dur := time.Now().Sub(now)
		l.Debug().Dur("duration", dur).Msg("Done fetching status")

		return nil
	}
}
