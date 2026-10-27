package health_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/health"
	"url-shortener/internal/http-server/handlers/health/mocks"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
)

func TestHealthHandler(t *testing.T) {
	cases := []struct {
		name      string
		respError string
		status    int
		pingError error
	}{
		{
			name:   "Success",
			status: http.StatusOK,
		},
		{
			name:      "Storage unavailable",
			respError: "storage is unavailable",
			status:    http.StatusServiceUnavailable,
			pingError: errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pingerMock := mocks.NewPinger(t)

			pingerMock.On("Ping").Return(tc.pingError).Once()

			handler := health.New(slogdiscard.NewDiscardLogger(), pingerMock)

			req, err := http.NewRequest(http.MethodGet, "/health", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			var resp response.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)
		})
	}
}
