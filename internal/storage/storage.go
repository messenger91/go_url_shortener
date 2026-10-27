// internal/storage/storage.go
package storage

import (
	"errors"
	"time"
)

var (
	ErrURLNotFound = errors.New("URL_NOT_FOUND_ERROR")
	ErrURLExists   = errors.New("URL_EXISTS_ERROR")
)

// URL is a stored short link. ExpiresAt is zero for links that never expire.
type URL struct {
	URL       string
	Alias     string
	ExpiresAt time.Time
}
