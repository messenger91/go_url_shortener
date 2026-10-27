package redirect_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/redirect/mocks"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func TestRedirectHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		expiresAt time.Time
		status    int
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			url:    "https://www.google.com/",
			status: http.StatusFound,
		},
		{
			name:      "Expired",
			alias:     "test_alias",
			url:       "https://www.google.com/",
			expiresAt: time.Now().Add(-time.Hour),
			status:    http.StatusGone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlGetterMock := mocks.NewURLGetter(t)

			urlGetterMock.On("GetURL", tc.alias).
				Return(storage.URL{
					URL:       tc.url,
					Alias:     tc.alias,
					ExpiresAt: tc.expiresAt,
				}, nil).Once()

			// visits are only recorded for non-expired links
			if tc.status == http.StatusFound {
				urlGetterMock.On("RecordVisit", tc.alias).
					Return(nil).Once()
			}

			r := chi.NewRouter()
			r.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

			ts := httptest.NewServer(r)
			defer ts.Close()

			if tc.status == http.StatusFound {
				redirectedToURL, err := api.GetRedirect(ts.URL + "/" + tc.alias)
				require.NoError(t, err)

				// Check the final URL after redirection.
				assert.Equal(t, tc.url, redirectedToURL)

				return
			}

			resp, err := http.Get(ts.URL + "/" + tc.alias)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, tc.status, resp.StatusCode)
		})
	}
}
