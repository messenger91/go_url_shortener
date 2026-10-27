package migrations

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
)

//go:embed *.sql
var files embed.FS

// Up applies all not yet applied migrations from the embedded *.sql files,
// ordered by file name. Each migration runs in its own transaction and is
// recorded in the migrations table. SQL must be idempotent (IF NOT EXISTS):
// databases created before version tracking have the schema but an empty
// migrations table.
func Up(db *sqlx.DB) error {
	const op = "storage.sqlite.migrations.Up"

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("%s: create migrations table: %w", op, err)
	}

	appliedMigrations, err := applied(db)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	entries, err := fs.ReadDir(files, ".") // sorted by file name
	if err != nil {
		return fmt.Errorf("%s: read embedded migrations: %w", op, err)
	}

	for _, entry := range entries {
		if appliedMigrations[entry.Name()] {
			continue
		}

		if err := apply(db, entry.Name()); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func apply(db *sqlx.DB, name string) error {
	const op = "storage.sqlite.migrations.apply"

	query, err := files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("%s: read migration: %w", op, err)
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("%s: begin transaction: %w", op, err)
	}

	if _, err := tx.Exec(string(query)); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("%s: execute migration: %w", op, err)
	}

	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("%s: record migration: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	return nil
}

func applied(db *sqlx.DB) (map[string]bool, error) {
	const op = "storage.sqlite.migrations.applied"

	var names []string
	if err := db.Select(&names, "SELECT name FROM migrations"); err != nil {
		return nil, fmt.Errorf("%s: select applied migrations: %w", op, err)
	}

	appliedMigrations := make(map[string]bool, len(names))
	for _, name := range names {
		appliedMigrations[name] = true
	}

	return appliedMigrations, nil
}
