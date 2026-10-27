package stats_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/stats"
	"url-shortener/internal/http-server/handlers/url/stats/mocks"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func TestStatsHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		visits    int64
		respError string
		status    int
		urlError  error
		visErr    error
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			url:    "https://example.com",
			visits: 42,
			status: http.StatusOK,
		},
		{
			name:      "URL not found",
			alias:     "missing_alias",
			respError: "not found",
			status:    http.StatusNotFound,
			urlError:  storage.ErrURLNotFound,
		},
		{
			name:      "GetURL Error",
			alias:     "test_alias",
			respError: "internal error",
			status:    http.StatusInternalServerError,
			urlError:  errors.New("unexpected error"),
		},
		{
			name:      "GetVisitsCount Error",
			alias:     "test_alias",
			url:       "https://example.com",
			respError: "internal error",
			status:    http.StatusInternalServerError,
			visErr:    errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			statsGetterMock := mocks.NewURLStatsGetter(t)

			statsGetterMock.On("GetURL", tc.alias).
				Return(tc.url, tc.urlError).Once()

			// the count is only requested for URLs that were found
			if tc.urlError == nil {
				statsGetterMock.On("GetVisitsCount", tc.alias).
					Return(tc.visits, tc.visErr).Once()
			}

			handler := stats.New(slogdiscard.NewDiscardLogger(), statsGetterMock)

			req, err := http.NewRequest(http.MethodGet, "/url/"+tc.alias+"/stats", nil)
			require.NoError(t, err)

			// the handler reads the alias from chi route params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			var resp response.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)
		})
	}
}
