package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/health"
	"url-shortener/internal/http-server/handlers/login"
	"url-shortener/internal/http-server/handlers/logout"
	"url-shortener/internal/http-server/handlers/redirect"
	urlDelete "url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/list"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/stats"
	urlUpdate "url-shortener/internal/http-server/handlers/url/update"
	mwAuth "url-shortener/internal/http-server/middleware/auth"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/http-server/middleware/ratelimit"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/sqlite"

	"golang.org/x/time/rate"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", cfg.Version),
	)
	log.Debug("debug messages are enabled")

	storage, err := sqlite.Connect(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		// the web UI authenticates with a session cookie, API clients with basic auth
		r.Use(mwAuth.SessionOrBasic(cfg.Password, "url-shortener", map[string]string{
			cfg.User: cfg.Password,
		}))

		r.Post("/", save.New(log, storage, cfg.AliasLength))
		r.Get("/", list.New(log, storage))
		r.Put("/{alias}", urlUpdate.New(log, storage))
		r.Delete("/{alias}", urlDelete.New(log, storage))
		r.Get("/{alias}/stats", stats.New(log, storage))
	})

	loginRateLimiter := ratelimit.New(log, rate.Limit(cfg.RateLimit), cfg.RateBurst)
	router.With(loginRateLimiter).Post("/login", login.New(log, cfg.User, cfg.Password, cfg.Password))
	router.Post("/logout", logout.New(log))

	router.Get("/", ui.New(staticFS))

	redirectRateLimiter := ratelimit.New(log, rate.Limit(cfg.RateLimit), cfg.RateBurst)
	router.With(redirectRateLimiter).Get("/{alias}", redirect.New(log, storage))

	// public and unlimited; also used by the container HEALTHCHECK
	router.Get("/health", health.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))

		return
	}

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
