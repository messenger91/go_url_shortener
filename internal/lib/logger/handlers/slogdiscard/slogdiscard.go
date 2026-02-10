package slogdiscard

import (
	"context"
	"log/slog"
)

type DiscardHandler struct{}

func NewDiscardLogger() *slog.Logger {
	return slog.New(NewDiscardHandler())
}

func NewDiscardHandler() *DiscardHandler {
	return &DiscardHandler{}
}
func (h *DiscardHandler) Handle(_ context.Context, _ slog.Record) error {
	// ignore records
	return nil
}

func (h *DiscardHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	// return handler
	return h
}

func (h *DiscardHandler) WithGroup(_ string) slog.Handler {
	// return handler
	return h
}

func (h *DiscardHandler) Enabled(_ context.Context, _ slog.Level) bool {
	// always return false
	return false
}
