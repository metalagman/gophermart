package app

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/pressly/goose/v3"
)

func applyMigrations(embedMigrations embed.FS, db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
