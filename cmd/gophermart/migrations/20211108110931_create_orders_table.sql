-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS "orders" (
   id uuid DEFAULT uuid_generate_v4() NOT NULL UNIQUE,
   external_id text NOT NULL UNIQUE,
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
   user_id uuid NOT NULL,
   status varchar(255) NOT NULL DEFAULT 'NEW',
   accrual DECIMAL,
   PRIMARY KEY(id),
   CONSTRAINT fk_user
       FOREIGN KEY(user_id)
           REFERENCES users(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "orders";
-- +goose StatementEnd
