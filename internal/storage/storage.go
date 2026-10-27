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

// VisitsByDay is the number of visits recorded on one UTC calendar day.
// Date has the YYYY-MM-DD form.
type VisitsByDay struct {
	Date  string
	Count int64
}

// URLListItem is a stored link with its total visit count, for listings.
type URLListItem struct {
	URL       string
	Alias     string
	ExpiresAt time.Time
	Visits    int64
}
