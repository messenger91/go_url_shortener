package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "url-shortener/internal/lib/api/response"
)

const (
	// CookieName is the session cookie the web UI authenticates with.
	CookieName = "session"
	// SessionTTL is how long a minted session stays valid.
	SessionTTL = 24 * time.Hour
)

// NewSession mints a signed session token "<expiry unix>.<nonce>.<hmac>".
// The secret is the configured basic auth password, so rotating the password
// invalidates all outstanding sessions.
func NewSession(secret string) (string, error) {
	const op = "auth.NewSession"

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("%s: generate nonce: %w", op, err)
	}

	value := fmt.Sprintf("%d.%s", time.Now().Add(SessionTTL).Unix(), hex.EncodeToString(nonce))

	return value + "." + sign(secret, value), nil
}

// ValidSession reports whether the token is well-formed, unexpired and
// properly signed with the secret.
func ValidSession(token, secret string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false
	}

	value := parts[0] + "." + parts[1]

	return subtle.ConstantTimeCompare([]byte(sign(secret, value)), []byte(parts[2])) == 1
}

func sign(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))

	return hex.EncodeToString(mac.Sum(nil))
}

// SessionOrBasic lets a valid session cookie through and falls back to basic
// auth otherwise. When the client attempted no basic auth at all, the 401 is
// sent without WWW-Authenticate so that browser fetch() from the web UI does
// not pop the native login dialog.
func SessionOrBasic(secret, realm string, users map[string]string) func(http.Handler) http.Handler {
	basic := middleware.BasicAuth(realm, users)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie(CookieName); err == nil && ValidSession(cookie.Value, secret) {
				next.ServeHTTP(w, r)

				return
			}

			if r.Header.Get("Authorization") != "" {
				basic(next).ServeHTTP(w, r)

				return
			}

			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Error("unauthorized"))
		})
	}
}
