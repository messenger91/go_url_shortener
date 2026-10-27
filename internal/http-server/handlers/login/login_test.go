package login_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/login"
	"url-shortener/internal/http-server/middleware/auth"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
)

func TestLoginHandler(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		respError string
		status    int
		session   bool
	}{
		{
			name:    "Success",
			body:    `{"user": "admin", "password": "admin"}`,
			status:  http.StatusOK,
			session: true,
		},
		{
			name:      "Wrong password",
			body:      `{"user": "admin", "password": "wrong"}`,
			respError: "invalid credentials",
			status:    http.StatusUnauthorized,
		},
		{
			name:      "Wrong user",
			body:      `{"user": "nobody", "password": "admin"}`,
			respError: "invalid credentials",
			status:    http.StatusUnauthorized,
		},
		{
			name:      "Empty body",
			body:      ``,
			respError: "failed to decode request",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Missing password",
			body:      `{"user": "admin"}`,
			respError: "field Password is a required field",
			status:    http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := login.New(slogdiscard.NewDiscardLogger(), "admin", "admin", "test-secret")

			req, err := http.NewRequest(http.MethodPost, "/login", io.NopCloser(strings.NewReader(tc.body)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			var resp response.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			cookies := rr.Result().Cookies()
			if tc.session {
				session := findCookie(t, cookies, auth.CookieName)
				require.NotEmpty(t, session.Value)
				require.True(t, session.HttpOnly)
			} else {
				require.Nil(t, findCookie(t, cookies, auth.CookieName))
			}
		})
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()

	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}

	return nil
}
