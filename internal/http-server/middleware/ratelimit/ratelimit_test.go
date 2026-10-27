package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"url-shortener/internal/http-server/middleware/ratelimit"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
)

func TestRateLimit(t *testing.T) {
	// 1 request per second with a burst of 3
	mw := ratelimit.New(slogdiscard.NewDiscardLogger(), rate.Limit(1), 3)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/some_alias", nil)
		req.RemoteAddr = ip + ":12345"

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		return rr.Code
	}

	// the burst is allowed, then requests are limited
	require.Equal(t, http.StatusOK, request("1.1.1.1"))
	require.Equal(t, http.StatusOK, request("1.1.1.1"))
	require.Equal(t, http.StatusOK, request("1.1.1.1"))
	require.Equal(t, http.StatusTooManyRequests, request("1.1.1.1"))

	// buckets are per IP
	require.Equal(t, http.StatusOK, request("2.2.2.2"))
}
