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

//nolint:funlen
func TestStatsHandler(t *testing.T) {
	cases := []struct {
		name        string
		alias       string
		url         string
		visits      int64
		days        int
		query       string
		byDay       []storage.VisitsByDay
		respError   string
		status      int
		invalidDays bool
		urlError    error
		visErr      error
		byDayErr    error
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			url:    "https://example.com",
			visits: 42,
			days:   30,
			byDay: []storage.VisitsByDay{
				{Date: "2026-10-27", Count: 7},
				{Date: "2026-10-26", Count: 3},
			},
			status: http.StatusOK,
		},
		{
			name:  "Custom days",
			alias: "test_alias",
			url:   "https://example.com",
			days:  5,
			query: "?days=5",
			byDay: []storage.VisitsByDay{
				{Date: "2026-10-27", Count: 1},
			},
			status: http.StatusOK,
		},
		{
			name:        "Invalid days",
			alias:       "test_alias",
			days:        30,
			query:       "?days=abc",
			respError:   "invalid days parameter",
			status:      http.StatusBadRequest,
			invalidDays: true,
		},
		{
			name:      "URL not found",
			alias:     "missing_alias",
			days:      30,
			respError: "not found",
			status:    http.StatusNotFound,
			urlError:  storage.ErrURLNotFound,
		},
		{
			name:      "GetURL Error",
			alias:     "test_alias",
			days:      30,
			respError: "internal error",
			status:    http.StatusInternalServerError,
			urlError:  errors.New("unexpected error"),
		},
		{
			name:      "GetVisitsCount Error",
			alias:     "test_alias",
			url:       "https://example.com",
			days:      30,
			respError: "internal error",
			status:    http.StatusInternalServerError,
			visErr:    errors.New("unexpected error"),
		},
		{
			name:      "GetVisitsByDay Error",
			alias:     "test_alias",
			url:       "https://example.com",
			days:      30,
			respError: "internal error",
			status:    http.StatusInternalServerError,
			byDayErr:  errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			statsGetterMock := mocks.NewURLStatsGetter(t)

			// the storage is not touched at all when the days parameter is invalid
			if !tc.invalidDays {
				statsGetterMock.On("GetURL", tc.alias).
					Return(storage.URL{URL: tc.url, Alias: tc.alias}, tc.urlError).Once()
			}

			// the counts are only requested for URLs that were found,
			// and not at all when the days parameter is invalid
			if tc.urlError == nil && !tc.invalidDays {
				statsGetterMock.On("GetVisitsCount", tc.alias).
					Return(tc.visits, tc.visErr).Once()
			}

			if tc.urlError == nil && tc.visErr == nil && !tc.invalidDays {
				statsGetterMock.On("GetVisitsByDay", tc.alias, tc.days).
					Return(tc.byDay, tc.byDayErr).Once()
			}

			handler := stats.New(slogdiscard.NewDiscardLogger(), statsGetterMock)

			req, err := http.NewRequest(http.MethodGet, "/url/"+tc.alias+"/stats"+tc.query, nil)
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

			var statsResp stats.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &statsResp))

			require.Len(t, statsResp.VisitsByDay, len(tc.byDay))
			for i, day := range tc.byDay {
				require.Equal(t, day.Date, statsResp.VisitsByDay[i].Date)
				require.Equal(t, day.Count, statsResp.VisitsByDay[i].Count)
			}
		})
	}
}
