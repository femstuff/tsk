package cache

import (
	"context"
	"time"
)

type NoopCache struct{}

func NewNoop() *NoopCache {
	return &NoopCache{}
}

func (c *NoopCache) Ping(_ context.Context) error {
	return nil
}

func (c *NoopCache) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, nil
}

func (c *NoopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (c *NoopCache) Delete(_ context.Context, _ string) error {
	return nil
}
