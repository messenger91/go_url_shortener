// internal/storage/storage.go
package storage

import "errors"

var (
	ErrURLNotFound = errors.New("URL_NOT_FOUND_ERROR")
	ErrURLExists   = errors.New("URL_EXISTS_ERROR")
)
