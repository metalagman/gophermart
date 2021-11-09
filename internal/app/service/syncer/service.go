package syncer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/google/uuid"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/model"
	"gophermart/pkg/accrual"
	"runtime"
	"sync"
	"time"
)

const (
	statusRegistered = "REGISTERED"
	statusInvalid    = "INVALID"
	statusProcessing = "PROCESSING"
	statusProcessed  = "PROCESSED"
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
	jobTimeout    time.Duration
}

func (s *Service) JobTimeout() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobTimeout
}

func (s *Service) SetJobTimeout(jobTimeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Lock()
	s.jobTimeout = jobTimeout
}

func New(db *sql.DB, ac *accrual.Service) (*Service, error) {
	s := &Service{
		logger: logger.Global().WithComponent("AccrualSync.Service"),

		jobs:    make(chan Job),
		stopCh:  make(chan struct{}),
		accrual: ac,
		db:      db,

		fetchInterval: 5 * time.Second,
		jobTimeout:    30 * time.Second,
	}
	s.Start(runtime.GOMAXPROCS(0) * 2)

	return s, nil
}

func (s *Service) Start(numWorkers int) {
	s.logger.Info().Int("worker_num", numWorkers).Msg("Starting workers")
	for i := 0; i < numWorkers; i++ {
		go func(workerID int, l logger.Logger, jobs chan Job, stop chan struct{}) {
			wl := l.With().Int("worker_id", workerID).Logger()
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
					jl := wl.With().Str("job_id", id.String()).Logger()
					jl.Info().Msg("Running job")

					err := retry.Retry(
						func(attempt uint) error {
							defer func() {
								v := recover()
								if v != nil {
									jl.Error().
										Uint("attempt", attempt).
										Str("panic", fmt.Sprintf("%v", v)).
										Msg("Panic recovered")
								}
							}()
							err := job()
							if err != nil {
								jl.Error().Err(err).
									Uint("attempt", attempt).
									Msg("Job run failed")
							}
							return err
						},
						strategy.Limit(3),
						strategy.Backoff(backoff.Fibonacci(10*time.Millisecond)),
					)
					if err != nil {
						jl.Error().Msg("Job completely failed")
						return
					}

					jl.Info().Msg("Job done")
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
				s.jobs <- s.FetchAll()
				t.Reset(fetchInterval)
			}
		}

	}(s.logger, s.fetchInterval)
}

func (s *Service) Stop() {
	s.logger.Debug().Msg("Service shutdown")
	close(s.stopCh)
}

func (s *Service) Run(job Job) {
	s.jobs <- job
}

func (s *Service) FetchOrderDetails(id uuid.UUID) Job {
	return func() error {
		l := s.logger.WithComponent("AccrualSync.Job.FetchOrderDetails")
		l.Debug().Msg("Fetching status")

		ctx, cancel := context.WithTimeout(context.Background(), s.JobTimeout())
		defer cancel()
		ctx = l.WithContext(ctx)

		l.Debug().Msg("Starting transaction")

		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
		})
		if err != nil {
			l.Error().Err(err).Msg("DB transaction begin")
			return err
		}

		l.Debug().Msg("Locking user balance")

		var oldStatus, externalID string
		var userID uuid.UUID
		const sqlLock = `SELECT status, external_id, user_id FROM orders WHERE id=$1 FOR UPDATE`
		if err := tx.QueryRowContext(ctx, sqlLock, id).Scan(&oldStatus, &externalID, &userID); err != nil {
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

		l.Debug().Msg("Updating order status")

		const sqlUpdate = `UPDATE orders SET status=$1, accrual=$2 WHERE id=$3`
		_, err = tx.ExecContext(ctx, sqlUpdate, out.Status, out.Accrual, id)
		if err != nil {
			l.Error().Err(err).Msg("Status update failed")
			_ = tx.Rollback()
			return err
		}

		if oldStatus != out.Status && out.Status == statusProcessed && out.Accrual.Valid {
			l.Debug().Msg("Updating balance")
			const sqlTx = `INSERT INTO transactions (type_id, user_id, order_id, external_order_id, amount) VALUES ($1, $2, $3, $4, $5)`
			_, err = tx.ExecContext(ctx, sqlTx, model.TransactionTypeReplenishment, userID, id, externalID, out.Accrual)
			if err != nil {
				l.Error().Err(err).Msg("TX insert failed")
				_ = tx.Rollback()
				return err
			}

			const sqlUpdateBalance = `UPDATE users SET balance=balance + $1 WHERE id = $2`
			_, err = tx.ExecContext(ctx, sqlUpdateBalance, out.Accrual, userID)
			if err != nil {
				l.Error().Err(err).Msg("Balance update failed")
				_ = tx.Rollback()
				return err
			}
		}

		l.Debug().Msg("Commit transaction")
		if err := tx.Commit(); err != nil {
			l.Error().Err(err).Msg("TX commit failed")
			return err
		}

		dur := time.Since(now)
		l.Debug().Dur("duration", dur).Msg("Done fetching status")

		return nil
	}
}

func (s *Service) FetchAll() Job {
	return func() error {
		l := s.logger.WithComponent("AccrualSync.Job.FetchAll")
		l.Debug().Msg("Fetching status")

		ctx, cancel := context.WithTimeout(context.Background(), s.JobTimeout())
		defer cancel()
		ctx = l.WithContext(ctx)

		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		})
		if err != nil {
			l.Error().Err(err).Msg("DB transaction begin")
			return err
		}
		defer func(tx *sql.Tx) {
			_ = tx.Rollback()
		}(tx)

		const sqlRead = `SELECT id FROM orders WHERE status in ($1, $2)`

		rows, err := tx.QueryContext(ctx, sqlRead, statusRegistered, statusProcessing)
		if err != nil {
			_ = tx.Rollback()
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			l.Error().Err(err).Msg("DB select")
			return fmt.Errorf("query context: %w", err)
		}
		defer func(rows *sql.Rows) {
			_ = rows.Close()
		}(rows)

		var id uuid.UUID
		for rows.Next() {
			if err := rows.Err(); err != nil {
				_ = tx.Rollback()
				l.Error().Err(err).Msg("rows.Next()")
				return fmt.Errorf("rows next: %w", err)
			}
			if err := rows.Scan(&id); err != nil {
				go s.Run(s.FetchOrderDetails(id))
			}
		}

		return nil
	}
}
