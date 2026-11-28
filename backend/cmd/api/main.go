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

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/comment"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/label"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/notification"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/org"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/email"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/security"
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

	deps, err := buildDeps(cfg, pool)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(deps),
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

// buildDeps lắp adapter hạ tầng vào use case (composition root).
func buildDeps(cfg config, pool *pgxpool.Pool) (httpapi.Deps, error) {
	jwt, err := security.NewJWT([]byte(cfg.JWTSecret), cfg.AccessTTL)
	if err != nil {
		return httpapi.Deps{}, fmt.Errorf("jwt: %w", err)
	}
	mailer, err := email.NewSMTPMailer(email.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		From:     cfg.SMTPFrom,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
	})
	if err != nil {
		return httpapi.Deps{}, fmt.Errorf("mailer: %w", err)
	}

	users := postgres.NewUserRepo(pool)
	now := time.Now
	projects := project.NewService(postgres.NewProjectRepo(pool), postgres.NewProjectTx(pool))
	settings := setting.NewService(postgres.NewSettingRepo(pool))
	// Hub trong bộ nhớ: đủ cho một tiến trình API (docs/decisions.md mục C, feature/kanban).
	hub := realtime.NewHub()
	return httpapi.Deps{
		DB:         pool,
		CORSOrigin: cfg.CORSOrigin,
		Auth:       auth.NewService(users, postgres.NewRefreshTokenRepo(pool), security.NewBcryptHasher(bcrypt.DefaultCost), jwt, now),
		Accounts:   account.NewService(users, mailer, cfg.AppBaseURL, now, slog.Default()),
		Org:        org.NewService(postgres.NewOrgRepo(pool)),
		Projects:   projects,
		Tasks:      task.NewService(postgres.NewTaskRepo(pool), postgres.NewTaskTx(pool), projects, settings, hub),
		Kanban:     kanban.NewService(postgres.NewTaskRepo(pool), postgres.NewKanbanTx(pool), projects, hub),
		// CF-07
		Subtasks:      subtask.NewService(postgres.NewSubtaskRepo(pool), postgres.NewSubtaskTx(pool), projects),
		Comments:      comment.NewService(postgres.NewCommentRepo(pool), postgres.NewCommentTx(pool), projects),
		Notifications: notification.NewService(postgres.NewNotificationRepo(pool)),
		Hub:           hub,
		// CF-08
		TimeLogs: timelog.NewService(postgres.NewTimeLogRepo(pool), postgres.NewTimeLogTx(pool), projects, hub, time.Now),
		// CF-09: nghiệm thu dùng lại luật chuyển cột của kanban.Service.
		Reviews:  kanban.NewReviewService(postgres.NewReviewRepo(pool), postgres.NewReviewTx(pool), projects, hub),
		Labels:   label.NewService(postgres.NewLabelRepo(pool)),
		Settings: settings,
		Tokens:   jwt,
		Now:      now,
	}, nil
}
