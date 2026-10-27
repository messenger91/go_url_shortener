package auth_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/middleware/auth"
)

const (
	secret = "test-secret"
	realm  = "test-realm"
)

// basicAuthHeader returns an Authorization header value for user:pass.
func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestSessionOrBasic(t *testing.T) {
	const (
		apiUser     = "admin"
		apiPassword = "admin"
	)

	newHandler := func(calls *int) http.Handler {
		return auth.SessionOrBasic(secret, realm, map[string]string{apiUser: apiPassword})(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				*calls++

				w.WriteHeader(http.StatusOK)
			}),
		)
	}

	do := func(h http.Handler, cookie, authorization string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/url", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
		}
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		return rr
	}

	t.Run("valid session cookie", func(t *testing.T) {
		token, err := auth.NewSession(secret)
		require.NoError(t, err)

		calls := 0
		rr := do(newHandler(&calls), token, "")

		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, calls)
	})

	t.Run("invalid session falls back to basic auth", func(t *testing.T) {
		calls := 0
		h := newHandler(&calls)

		rr := do(h, "tampered.token.sig", basicAuthHeader(apiUser, apiPassword))
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, calls)

		rr = do(h, "tampered.token.sig", basicAuthHeader(apiUser, "wrong"))
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.NotEmpty(t, rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("no session and no basic auth answers json 401", func(t *testing.T) {
		calls := 0
		rr := do(newHandler(&calls), "", "")

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Equal(t, 0, calls)
		// no WWW-Authenticate, so the browser does not pop the native dialog
		require.Empty(t, rr.Header().Get("WWW-Authenticate"))
	})

	t.Run("expired session is rejected", func(t *testing.T) {
		calls := 0
		// expiry 1 is in the past, the signature never gets checked
		rr := do(newHandler(&calls), "1.aa.bb", "")

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Equal(t, 0, calls)
	})

	t.Run("session signed with another secret is rejected", func(t *testing.T) {
		otherToken, err := auth.NewSession("some-other-secret")
		require.NoError(t, err)

		calls := 0
		rr := do(newHandler(&calls), otherToken, "")

		require.Equal(t, http.StatusUnauthorized, rr.Code)
		require.Equal(t, 0, calls)
	})
}
