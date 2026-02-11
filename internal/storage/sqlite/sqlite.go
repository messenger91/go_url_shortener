// internal/storage/sqlite/sqlite.go
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/sqlite/migrations"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sqlx.DB
}

func Connect(path string) (*Storage, error) {
	// current func name for logs and errors
	const op = "storage.sqlite.Connect"

	db, err := sqlx.Connect("sqlite", path)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// init migrations
	migrations.Up(db)

	return &Storage{db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	// insert url
	stmt, err := s.db.Prepare("INSERT INTO url(url, alias) VALUES(?, ?)")

	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	// exec query
	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {

		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var resURL string

	err = stmt.QueryRow(alias).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	}

	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) DeleteURL(alias string) (string, error) {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare("DELETE FROM url WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var resURL string

	err = stmt.QueryRow(alias).Scan(&resURL)

	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}
