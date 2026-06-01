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
	"tsk/backend-api/internal/application/estimateintent"
	"tsk/backend-api/internal/infrastructure/cache"
	"tsk/backend-api/internal/infrastructure/config"
	"tsk/backend-api/internal/infrastructure/metrics"
	"tsk/backend-api/internal/infrastructure/persistence/postgres"
	prom "tsk/backend-api/internal/infrastructure/prometheus"
	"tsk/backend-api/internal/infrastructure/storage/localfs"
	"tsk/backend-api/internal/infrastructure/whisper"
	"tsk/backend-api/internal/integrations/bitrixclient"
	"tsk/backend-api/internal/integrations/bitrixoauth"
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

	if err := postgres.RunMigrations(ctx, store.Pool()); err != nil {
		logger.Error("failed to apply postgres migrations", "error", err)
		os.Exit(1)
	}

	cacheStore, closeCache, err := cache.NewFromConfig(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := closeCache(); closeErr != nil {
			logger.Error("failed to close redis", "error", closeErr)
		}
	}()

	storage := localfs.New(cfg.StorageRoot)

	var whisperClient *whisper.Client
	if strings.TrimSpace(cfg.WhisperBaseURL) != "" {
		whisperClient = whisper.New(cfg.WhisperBaseURL, 4*time.Minute)
	}
	bitrixHTTP := &http.Client{Timeout: 90 * time.Second}
	bitrixClient := bitrixclient.New(cfg.BitrixWebhookURL, bitrixHTTP)

	portalDomain := strings.TrimSpace(cfg.BitrixPortalDomain)
	if portalDomain == "" {
		portalDomain = bitrixoauth.PortalDomainFromWebhook(cfg.BitrixWebhookURL)
	}
	oauthCfg := bitrixoauth.Config{
		ClientID:     cfg.BitrixOAuthClientID,
		ClientSecret: cfg.BitrixOAuthClientSecret,
		PortalDomain: portalDomain,
		RedirectURI:  cfg.BitrixOAuthRedirectURI,
		MobileScheme: cfg.BitrixMobileAppScheme,
	}

	service := app.NewService(
		store,
		store,
		store,
		store,
		store,
		store,
		store,
		storage,
		cacheStore,
		cfg.StorageRoot,
		collector,
		app.IntegrationsConfig{
			BitrixWebhookURL:        cfg.BitrixWebhookURL,
			BitrixDealEstimateField: cfg.BitrixDealEstimateField,
			ApprovalEmail:           cfg.ApprovalEmail,
			BitrixOAuth:             oauthCfg,
		},
		whisperClient,
		bitrixClient,
		store,
	)
	if enricher := estimateintent.NewLLMEnricher(cfg.LLMAPIURL, cfg.LLMAPIKey, cfg.LLMModel, &http.Client{Timeout: 90 * time.Second}); enricher != nil {
		service.SetEstimateLLM(enricher)
		logger.Info("estimate LLM enricher enabled", "model", cfg.LLMModel, "url", cfg.LLMAPIURL)
	} else {
		logger.Info("estimate LLM enricher disabled (set LLM_API_KEY to enable)")
	}

	if err := service.EnsureSeedData(ctx); err != nil {
		logger.Error("failed to ensure seed data", "error", err)
		os.Exit(1)
	}
	if err := service.EnsureRuntimeSettings(ctx); err != nil {
		logger.Error("failed to ensure runtime settings", "error", err)
		os.Exit(1)
	}

	recovered, err := service.RecoverStuckJobs(ctx, cfg.JobRecoveryInterval)
	if err != nil {
		logger.Error("failed to recover stuck jobs", "error", err)
		os.Exit(1)
	}
	if recovered > 0 {
		logger.Warn("recovered stuck document jobs", "count", recovered)
	}

	processor := app.NewProcessor(service, logger, cfg.ProcessingInterval)
	prometheusClient := prom.NewClient(cfg.PrometheusURL, &http.Client{Timeout: 2 * time.Second})
	handler := httpapi.NewHandler(service, cfg, collector, prometheusClient, httpapi.HealthDeps{
		Platform:    store,
		Cache:       cacheStore,
		StorageRoot: cfg.StorageRoot,
	})
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
		"redis", strings.TrimSpace(cfg.RedisURL) != "",
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
