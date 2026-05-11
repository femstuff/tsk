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
	defer ticker.Stop()

	p.processUntilIdle(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processUntilIdle(ctx)
		}
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
