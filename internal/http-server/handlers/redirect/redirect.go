package redirect

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

//go:generate go run github.com/vektra/mockery/v2@v2.53.8 --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (storage.URL, error)
	RecordVisit(alias string) error
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

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

		storedURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)

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

		if !storedURL.ExpiresAt.IsZero() && time.Now().After(storedURL.ExpiresAt) {
			log.Info("url expired",
				slog.String("alias", alias),
				slog.Time("expires_at", storedURL.ExpiresAt),
			)

			render.Status(r, http.StatusGone)
			render.JSON(w, r, resp.Error("url expired"))

			return
		}

		log.Info("got url", slog.String("url", storedURL.URL))

		// a failed visit record must not break the redirect
		if err := urlGetter.RecordVisit(alias); err != nil {
			log.Error("failed to record visit", sl.Err(err))
		}

		// redirect to found url
		http.Redirect(w, r, storedURL.URL, http.StatusFound)
	}
}
