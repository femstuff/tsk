package documentjob

import (
	"context"
	"log/slog"
	"time"
)

type Processor struct {
	service  *Service
	logger   *slog.Logger
	interval time.Duration
}

func NewProcessor(service *Service, logger *slog.Logger, interval time.Duration) *Processor {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	return &Processor{
		service:  service,
		logger:   logger,
		interval: interval,
	}
}

func (p *Processor) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	metricsTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer metricsTicker.Stop()

	p.processUntilIdle(ctx)
	p.syncMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processUntilIdle(ctx)
		case <-metricsTicker.C:
			p.syncMetrics(ctx)
		}
	}
}

func (p *Processor) syncMetrics(ctx context.Context) {
	if err := p.service.syncJobStatusMetrics(ctx); err != nil {
		p.logger.Warn("failed to sync job metrics", "error", err)
	}
}

func (p *Processor) processUntilIdle(ctx context.Context) {
	for {
		processed, err := p.service.ProcessNextQueuedJob(ctx)
		if err != nil {
			p.logger.Error("document processor cycle failed", "error", err)
			return
		}

		if !processed {
			return
		}
	}
}
