package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	pg "github.com/lib/pq"
	"gophermart/internal/app/model"
	"reflect"
	"testing"
)

func TestUserRepository_Create(t *testing.T) {
	mdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	newUUID := uuid.New()

	mock.ExpectQuery(`INSERT INTO users`).WithArgs("Good", "Password").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(newUUID.String()),
	)
	mock.ExpectQuery(`INSERT INTO users`).WithArgs("Existing", "Password").WillReturnError(
		&pg.Error{
			Code:    pgerrcode.IntegrityConstraintViolation,
			Message: "some error",
		})
	mock.ExpectQuery(`INSERT INTO users`).WithArgs("Failing", "Password").WillReturnError(
		errors.New("you shall not pass"),
	)
	defer func() {
		_ = mdb.Close()
	}()

	type args struct {
		ctx  context.Context
		user *model.User
	}
	tests := []struct {
		name    string
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "create user",
			args: args{
				context.TODO(),
				&model.User{
					Name:     "Good",
					Password: "Password",
				},
			},
			want: &model.User{
				ID:       newUUID,
				Name:     "Good",
				Password: "Password",
			},
			wantErr: false,
		},
		{
			name: "create existing user",
			args: args{
				context.TODO(),
				&model.User{
					Name:     "Existing",
					Password: "Password",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "create failing user",
			args: args{
				context.TODO(),
				&model.User{
					Name:     "Failing",
					Password: "Password",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserRepository{
				db: mdb,
			}
			got, err := r.Create(tt.args.ctx, tt.args.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserRepository_Read(t *testing.T) {
	mdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	goodUUID := uuid.New()
	missingUUID := uuid.New()
	failingUUID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs(goodUUID.String()).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow(goodUUID.String(), "Good"),
	)
	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs(missingUUID.String()).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs(failingUUID.String()).WillReturnError(
		errors.New("you shall not pass"),
	)
	defer func() {
		_ = mdb.Close()
	}()

	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "read good user",
			args: args{
				context.TODO(),
				goodUUID,
			},
			want: &model.User{
				ID:   goodUUID,
				Name: "Good",
			},
			wantErr: false,
		},
		{
			name: "read missing user",
			args: args{
				context.TODO(),
				missingUUID,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "read failing user",
			args: args{
				context.TODO(),
				failingUUID,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserRepository{
				db: mdb,
			}
			got, err := r.Read(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Read() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserRepository_ReadByNameAndPassword(t *testing.T) {
	mdb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	goodUUID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs("Good", "Password").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow(goodUUID.String(), "Good"),
	)
	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs("Good", "BadPassword").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT (.+) FROM users`).WithArgs("Failing", "Password").WillReturnError(
		errors.New("you shall not pass"),
	)
	defer func() {
		_ = mdb.Close()
	}()

	type args struct {
		ctx      context.Context
		name     string
		password string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "read with ok password",
			args: args{
				context.TODO(),
				"Good",
				"Password",
			},
			want: &model.User{
				ID:   goodUUID,
				Name: "Good",
			},
			wantErr: false,
		},
		{
			name: "read with bad password",
			args: args{
				context.TODO(),
				"Good",
				"BadPassword",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "read failing user",
			args: args{
				context.TODO(),
				"Failing",
				"BadPassword",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UserRepository{
				db: mdb,
			}
			got, err := r.ReadByNameAndPassword(tt.args.ctx, tt.args.name, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadByNameAndPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadByNameAndPassword() got = %v, want %v", got, tt.want)
			}
		})
	}
}
