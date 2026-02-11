package migrations

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TODO:
func Up(db *sqlx.DB) (*sql.Rows, error) {
	op := "storage.sqlite.Migration"
	query := `
	
	CREATE TABLE IF NOT EXISTS migrations (id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
	);

	-- 1770783317_create_url
	CREATE TABLE IF NOT EXISTS url (id INTEGER PRIMARY KEY AUTOINCREMENT,
		url VARCHAR(255) NOT NULL,
		alias VARCHAR(255) NOT NULL UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`

	result, err := db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}
