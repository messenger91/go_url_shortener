package ratelimit

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/render"
	"golang.org/x/time/rate"

	resp "url-shortener/internal/lib/api/response"
)

const (
	visitorTTL    = 2 * time.Minute
	cleanupPeriod = 30 * time.Second
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// New returns a per-IP token bucket middleware. Idle visitors are dropped by
// a background goroutine that lives as long as the process.
func New(log *slog.Logger, rps rate.Limit, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex

	visitors := make(map[string]*visitor)

	go func() {
		ticker := time.NewTicker(cleanupPeriod)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > visitorTTL {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.Lock()
			v, ok := visitors[ip]
			if !ok {
				v = &visitor{limiter: rate.NewLimiter(rps, burst)}
				visitors[ip] = v
			}
			v.lastSeen = time.Now()
			mu.Unlock()

			if !v.limiter.Allow() {
				log.Info("rate limit exceeded", slog.String("ip", ip))

				render.Status(r, http.StatusTooManyRequests)
				render.JSON(w, r, resp.Error("too many requests"))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
