package cache

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Cache interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

const (
	KeyTemplatesList = "tsk:cache:templates:list"
)

func BitrixTasksKey(responsibleID int, limit int) string {
	return fmt.Sprintf("tsk:cache:bitrix:tasks:%d:%d", responsibleID, limit)
}

func BitrixDealsKey(userID int, limit int, search string) string {
	return fmt.Sprintf("tsk:cache:bitrix:deals:%d:%d:%s", userID, limit, strings.TrimSpace(strings.ToLower(search)))
}
