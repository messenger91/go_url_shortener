package stats

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.8 --name=URLStatsGetter
type URLStatsGetter interface {
	GetURL(alias string) (storage.URL, error)
	GetVisitsCount(alias string) (int64, error)
}

type Response struct {
	resp.Response
	Alias     string     `json:"alias,omitempty"`
	URL       string     `json:"url,omitempty"`
	Visits    int64      `json:"visits"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func New(log *slog.Logger, urlStatsGetter URLStatsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.stats.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid request"))

			return
		}

		storedURL, err := urlStatsGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", slog.String("alias", alias))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to get url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		visits, err := urlStatsGetter.GetVisitsCount(alias)
		if err != nil {
			log.Error("failed to get visits count", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		log.Info("got url stats", slog.String("alias", alias), slog.Int64("visits", visits))

		response := Response{
			Response: resp.OK(),
			Alias:    alias,
			URL:      storedURL.URL,
			Visits:   visits,
		}

		if !storedURL.ExpiresAt.IsZero() {
			response.ExpiresAt = &storedURL.ExpiresAt
		}

		render.JSON(w, r, response)
	}
}
