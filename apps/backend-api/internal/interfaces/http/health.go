package httpapi

import (
	"context"
	"os"

	platform "tsk/backend-api/internal/domain/platform"
	"tsk/backend-api/internal/infrastructure/cache"
)

type HealthDeps struct {
	Platform    platform.Repository
	Cache       cache.Cache
	StorageRoot string
}

func (d HealthDeps) Check(ctx context.Context) map[string]string {
	checks := map[string]string{
		"database": "ok",
		"redis":    "ok",
		"storage":  "ok",
	}

	if d.Platform == nil {
		checks["database"] = "not_configured"
	} else if err := d.Platform.Ping(ctx); err != nil {
		checks["database"] = err.Error()
	}

	if d.Cache == nil {
		checks["redis"] = "not_configured"
	} else if _, ok := d.Cache.(*cache.NoopCache); ok {
		checks["redis"] = "disabled"
	} else if err := d.Cache.Ping(ctx); err != nil {
		checks["redis"] = err.Error()
	}

	if d.StorageRoot == "" {
		checks["storage"] = "not_configured"
	} else if info, err := os.Stat(d.StorageRoot); err != nil {
		checks["storage"] = err.Error()
	} else if !info.IsDir() {
		checks["storage"] = "not_a_directory"
	}

	return checks
}

func (d HealthDeps) IsHealthy(checks map[string]string) bool {
	for _, status := range checks {
		if status != "ok" && status != "disabled" {
			return false
		}
	}

	return true
}
