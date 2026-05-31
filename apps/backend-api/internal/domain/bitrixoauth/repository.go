package bitrixoauth

import (
	"context"
	"time"
)

type Repository interface {
	CreateSession(ctx context.Context, params SessionCreateParams) (Session, error)
	GetSessionByID(ctx context.Context, id string) (Session, error)
	GetSessionByState(ctx context.Context, state string) (Session, error)
	ActivateSession(ctx context.Context, id string, params SessionActivateParams) (Session, error)
	UpdateSessionTokens(ctx context.Context, id string, params SessionTokenUpdateParams) (Session, error)
	ListSessions(ctx context.Context, limit int) ([]Session, error)
	RevokeSession(ctx context.Context, id string, updatedAt time.Time) error
}
