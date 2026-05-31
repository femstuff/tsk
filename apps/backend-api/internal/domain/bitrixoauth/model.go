package bitrixoauth

import (
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("bitrix oauth session not found")

const (
	StatusPending = "pending"
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

type Session struct {
	ID           string
	State        string
	Status       string
	PortalDomain string
	RestEndpoint string
	OAuthScopes  string
	BitrixUserID int
	UserName     string
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SessionCreateParams struct {
	ID           string
	State        string
	Status       string
	PortalDomain string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SessionActivateParams struct {
	PortalDomain string
	RestEndpoint string
	OAuthScopes  string
	BitrixUserID int
	UserName     string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UpdatedAt    time.Time
}

type SessionTokenUpdateParams struct {
	AccessToken  string
	RefreshToken string
	RestEndpoint string
	OAuthScopes  string
	ExpiresAt    time.Time
	UpdatedAt    time.Time
}
