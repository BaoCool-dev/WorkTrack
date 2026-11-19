// Command api chạy WorkTrack HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/transport/httpapi"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(httpapi.ContextHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("api stopped with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	if err := pool.Ping(pingCtx); err != nil {
		// Không dừng: /readyz sẽ báo trạng thái thật, container tự hồi khi DB lên.
		slog.Warn("database not reachable at startup", "err", err)
	}
	cancel()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(httpapi.Deps{DB: pool, CORSOrigin: cfg.CORSOrigin}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	slog.Info("shutdown complete")
	return nil
}
