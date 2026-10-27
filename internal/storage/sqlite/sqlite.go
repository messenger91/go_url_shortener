// internal/storage/sqlite/sqlite.go
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/sqlite/migrations"

	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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

	// apply migrations
	if err := migrations.Up(db); err != nil {
		return nil, fmt.Errorf("%s: apply migrations: %w", op, err)
	}

	return &Storage{db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string, expiresAt time.Time) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	// insert url
	stmt, err := s.db.Prepare("INSERT INTO url(url, alias, expires_at) VALUES(?, ?, ?)")

	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	// a zero time means the link never expires and must be stored as NULL
	var expiresArg any
	if !expiresAt.IsZero() {
		expiresArg = expiresAt
	}

	// exec query
	res, err := stmt.Exec(urlToSave, alias, expiresArg)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (storage.URL, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url, expires_at FROM url WHERE alias = ?")
	if err != nil {
		return storage.URL{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var (
		resURL      string
		expiresNull sql.NullTime
	)

	err = stmt.QueryRow(alias).Scan(&resURL, &expiresNull)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.URL{}, storage.ErrURLNotFound
	}

	if err != nil {
		return storage.URL{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	res := storage.URL{
		URL:   resURL,
		Alias: alias,
	}

	if expiresNull.Valid {
		res.ExpiresAt = expiresNull.Time
	}

	return res, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare("DELETE FROM url WHERE alias = ?")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	res, err := stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: get affected rows: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}

	return nil
}

func (s *Storage) RecordVisit(alias string) error {
	const op = "storage.sqlite.RecordVisit"

	stmt, err := s.db.Prepare("INSERT INTO visits(alias) VALUES(?)")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	if _, err := stmt.Exec(alias); err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return nil
}

func (s *Storage) GetVisitsCount(alias string) (int64, error) {
	const op = "storage.sqlite.GetVisitsCount"

	stmt, err := s.db.Prepare("SELECT COUNT(*) FROM visits WHERE alias = ?")
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var count int64

	err = stmt.QueryRow(alias).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return count, nil
}

// GetVisitsByDay returns the per-day visit counts for the last days calendar
// days, UTC-based, oldest first. Days without visits are reported with zero
// counts, so the result always has exactly days entries.
func (s *Storage) GetVisitsByDay(alias string, days int) ([]storage.VisitsByDay, error) {
	const op = "storage.sqlite.GetVisitsByDay"

	stmt, err := s.db.Prepare(`
		SELECT date(created_at), COUNT(*)
		FROM visits
		WHERE alias = ? AND created_at >= date('now', ?)
		GROUP BY date(created_at)
		ORDER BY date(created_at)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	rows, err := stmt.Query(alias, fmt.Sprintf("-%d days", days-1))
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int64)
	for rows.Next() {
		var (
			day   string
			count int64
		)

		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}

		counts[day] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: iterate rows: %w", op, err)
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	res := make([]storage.VisitsByDay, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")

		res = append(res, storage.VisitsByDay{
			Date:  day,
			Count: counts[day],
		})
	}

	return res, nil
}

func (s *Storage) Ping() error {
	const op = "storage.sqlite.Ping"

	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure,
// e.g. saving a URL with an alias that already exists.
func isUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}

	// SQLITE_CONSTRAINT (19); extended result codes keep the base code in the
	// low byte, e.g. SQLITE_CONSTRAINT_UNIQUE (2067).
	if sqliteErr.Code()&0xff != sqlite3.SQLITE_CONSTRAINT {
		return false
	}

	return strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed")
}
