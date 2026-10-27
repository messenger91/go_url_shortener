package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

//nolint:funlen
func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		ttl       int
		respError string
		status    int
		mockError error
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			url:    "https://google.com",
			status: http.StatusOK,
		},
		{
			name:   "Empty alias",
			alias:  "",
			url:    "https://google.com",
			status: http.StatusOK,
		},
		{
			name:   "With TTL",
			alias:  "test_alias",
			url:    "https://google.com",
			ttl:    3600,
			status: http.StatusOK,
		},
		{
			name:      "Empty URL",
			url:       "",
			alias:     "some_alias",
			respError: "field URL is a required field",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Invalid URL",
			url:       "some invalid URL",
			alias:     "some_alias",
			respError: "field URL is not a valid URL",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Invalid TTL",
			url:       "https://google.com",
			alias:     "test_alias",
			ttl:       -5,
			respError: "field TTL must be greater than 0",
			status:    http.StatusBadRequest,
		},
		{
			name:      "SaveURL Error",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "failed to add url",
			status:    http.StatusInternalServerError,
			mockError: errors.New("unexpected error"),
		},
		{
			name:      "SaveURL URL Exists",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "url already exists",
			status:    http.StatusConflict,
			mockError: storage.ErrURLExists,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
					Return(int64(1), tc.mockError).
					Once()
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, 8)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s", "ttl_seconds": %d}`, tc.url, tc.alias, tc.ttl)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

		})
	}
}
