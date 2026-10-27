package health

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.8 --name=Pinger
type Pinger interface {
	Ping() error
}

func New(log *slog.Logger, pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.health.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if err := pinger.Ping(); err != nil {
			log.Error("storage is unavailable", sl.Err(err))

			render.Status(r, http.StatusServiceUnavailable)
			render.JSON(w, r, resp.Error("storage is unavailable"))

			return
		}

		render.JSON(w, r, resp.OK())
	}
}
