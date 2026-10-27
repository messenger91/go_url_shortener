package update_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/http-server/handlers/url/update/mocks"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

//nolint:funlen
func TestUpdateHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		body      string
		respError string
		status    int
		mockError error
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			body:   `{"url": "https://example.com/updated"}`,
			status: http.StatusOK,
		},
		{
			name:   "Success with TTL",
			alias:  "test_alias",
			body:   `{"url": "https://example.com/updated", "ttl_seconds": 60}`,
			status: http.StatusOK,
		},
		{
			name:      "URL not found",
			alias:     "missing_alias",
			body:      `{"url": "https://example.com/updated"}`,
			respError: "not found",
			status:    http.StatusNotFound,
			mockError: storage.ErrURLNotFound,
		},
		{
			name:      "Update error",
			alias:     "test_alias",
			body:      `{"url": "https://example.com/updated"}`,
			respError: "internal error",
			status:    http.StatusInternalServerError,
			mockError: errors.New("unexpected error"),
		},
		{
			name:      "Invalid URL",
			alias:     "test_alias",
			body:      `{"url": "some invalid URL"}`,
			respError: "field URL is not a valid URL",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Invalid TTL",
			alias:     "test_alias",
			body:      `{"url": "https://example.com/updated", "ttl_seconds": -1}`,
			respError: "field TTL must be greater than 0",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Empty body",
			alias:     "test_alias",
			body:      ``,
			respError: "empty request",
			status:    http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlUpdaterMock := mocks.NewURLUpdater(t)

			// the storage is only reached with a valid decoded request
			if tc.respError == "" || tc.mockError != nil {
				urlUpdaterMock.On("UpdateURL", tc.alias, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
					Return(tc.mockError).
					Once()
			}

			handler := update.New(slogdiscard.NewDiscardLogger(), urlUpdaterMock)

			req, err := http.NewRequest(http.MethodPut, "/url/"+tc.alias, io.NopCloser(strings.NewReader(tc.body)))
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
