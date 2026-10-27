package list

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.8 --name=URLLister
type URLLister interface {
	ListURLs() ([]storage.URLListItem, error)
}

type item struct {
	Alias     string     `json:"alias"`
	URL       string     `json:"url"`
	Visits    int64      `json:"visits"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Response struct {
	resp.Response
	Items []item `json:"items"`
}

func New(log *slog.Logger, urlLister URLLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.list.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		listItems, err := urlLister.ListURLs()
		if err != nil {
			log.Error("failed to list urls", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		log.Info("listed urls", slog.Int("count", len(listItems)))

		response := Response{
			Response: resp.OK(),
			Items:    make([]item, 0, len(listItems)),
		}

		for _, listItem := range listItems {
			it := item{
				Alias:  listItem.Alias,
				URL:    listItem.URL,
				Visits: listItem.Visits,
			}

			if !listItem.ExpiresAt.IsZero() {
				expiresAt := listItem.ExpiresAt
				it.ExpiresAt = &expiresAt
			}

			response.Items = append(response.Items, it)
		}

		render.JSON(w, r, response)
	}
}
