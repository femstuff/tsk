package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	app "tsk/backend-api/internal/application/documentjob"
	"tsk/backend-api/internal/infrastructure/config"
	"tsk/backend-api/internal/infrastructure/metrics"
	"tsk/backend-api/internal/infrastructure/persistence/postgres"
	"tsk/backend-api/internal/infrastructure/storage/localfs"
	"tsk/backend-api/internal/infrastructure/whisper"
	"tsk/backend-api/internal/integrations/bitrixclient"
	httpapi "tsk/backend-api/internal/interfaces/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	collector := metrics.NewCollector()
	store, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := postgres.ApplySchema(ctx, store.Pool()); err != nil {
		logger.Error("failed to apply postgres schema", "error", err)
		os.Exit(1)
	}

	storage := localfs.New(cfg.StorageRoot)

	var whisperClient *whisper.Client
	if strings.TrimSpace(cfg.WhisperBaseURL) != "" {
		whisperClient = whisper.New(cfg.WhisperBaseURL, 4*time.Minute)
	}
	bitrixHTTP := &http.Client{Timeout: 90 * time.Second}
	bitrixClient := bitrixclient.New(cfg.BitrixWebhookURL, bitrixHTTP)

	service := app.NewService(
		store,
		store,
		store,
		store,
		store,
		store,
		storage,
		collector,
		app.IntegrationsConfig{
			BitrixWebhookURL: cfg.BitrixWebhookURL,
			ApprovalEmail:    cfg.ApprovalEmail,
		},
		whisperClient,
		bitrixClient,
	)
	if err := service.EnsureSeedData(ctx); err != nil {
		logger.Error("failed to ensure seed data", "error", err)
		os.Exit(1)
	}

	processor := app.NewProcessor(service, logger, cfg.ProcessingInterval)
	handler := httpapi.NewHandler(service, cfg, collector)
	router := httpapi.NewRouter(handler, collector)

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}

	logger.Info("starting backend API",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"addr", cfg.Address(),
	)

	go processor.Run(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("graceful shutdown failed", "error", shutdownErr)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("backend API stopped", "error", err)
		os.Exit(1)
	}
}
