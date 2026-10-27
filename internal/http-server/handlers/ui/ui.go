package ui

import (
	"io/fs"
	"net/http"
)

// New returns a handler serving the embedded index.html.
func New(static fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		index, err := fs.ReadFile(static, "index.html")
		if err != nil {
			http.Error(w, "ui is not available", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}
}
