package documentjob

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	bdomain "tsk/backend-api/internal/domain/bitrixoauth"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

const bitrixSessionHeader = "X-TSK-Bitrix-Session"

type BitrixOAuthStartResult struct {
	AuthorizeURL string `json:"authorizeUrl"`
	SessionID    string `json:"sessionId"`
	State        string `json:"state"`
}

type BitrixOAuthSessionView struct {
	SessionID       string `json:"sessionId"`
	Connected       bool   `json:"connected"`
	BitrixUserID    int    `json:"bitrixUserId"`
	UserName        string `json:"userName"`
	PortalDomain    string `json:"portalDomain"`
	OAuthScopes     string `json:"oauthScopes,omitempty"`
	TaskScopeGranted bool  `json:"taskScopeGranted"`
	AuthMode        string `json:"authMode"`
}

func (s *Service) BitrixOAuthEnabled() bool {
	return s.bitrixOAuth != nil && s.bitrixOAuthCfg.Enabled()
}

func (s *Service) StartBitrixOAuth(ctx context.Context) (BitrixOAuthStartResult, error) {
	if !s.BitrixOAuthEnabled() {
		return BitrixOAuthStartResult{}, errors.New("Bitrix OAuth не настроен на сервере (BITRIX_OAUTH_CLIENT_ID, BITRIX_OAUTH_CLIENT_SECRET, BITRIX_OAUTH_REDIRECT_URI)")
	}
	if s.bitrixOAuthSessions == nil {
		return BitrixOAuthStartResult{}, errors.New("oauth session storage is not configured")
	}

	now := s.now().UTC()
	sessionID := "bos-" + uuid.NewString()
	state := uuid.NewString()

	_, err := s.bitrixOAuthSessions.CreateSession(ctx, bdomain.SessionCreateParams{
		ID:           sessionID,
		State:        state,
		Status:       bdomain.StatusPending,
		PortalDomain: s.bitrixOAuthCfg.PortalDomain,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return BitrixOAuthStartResult{}, err
	}

	return BitrixOAuthStartResult{
		AuthorizeURL: s.bitrixOAuth.AuthorizeURL(state),
		SessionID:    sessionID,
		State:        state,
	}, nil
}

func (s *Service) CompleteBitrixOAuthCallback(ctx context.Context, code, state, callbackDomain, callbackScope string) (BitrixOAuthSessionView, error) {
	if !s.BitrixOAuthEnabled() {
		return BitrixOAuthSessionView{}, errors.New("Bitrix OAuth не настроен")
	}
	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)
	if code == "" || state == "" {
		return BitrixOAuthSessionView{}, errors.New("code и state обязательны")
	}

	pending, err := s.bitrixOAuthSessions.GetSessionByState(ctx, state)
	if err != nil {
		return BitrixOAuthSessionView{}, err
	}
	if pending.Status != bdomain.StatusPending {
		return BitrixOAuthSessionView{}, errors.New("oauth state уже использован")
	}

	token, err := s.bitrixOAuth.ExchangeCode(ctx, code)
	if err != nil {
		return BitrixOAuthSessionView{}, fmt.Errorf("обмен code на token: %w", err)
	}

	restEndpoint := strings.TrimSpace(token.ClientEndpoint)
	portal := bitrixclient.ResolvePortalHost(callbackDomain, pending.PortalDomain, restEndpoint, token.Domain)
	oauthScopes := strings.TrimSpace(callbackScope)
	if oauthScopes == "" {
		oauthScopes = strings.TrimSpace(token.Scope)
	}

	userID := token.UserID
	userName := ""
	if userID <= 0 || userName == "" {
		uid, name, userErr := s.bitrixOAuth.CurrentUser(ctx, portal, restEndpoint, token.AccessToken)
		if userErr == nil {
			if userID <= 0 {
				userID = uid
			}
			userName = name
		}
	}

	now := s.now().UTC()
	expiresAt := now.Add(time.Duration(token.ExpiresIn) * time.Second)
	if token.ExpiresIn <= 0 {
		expiresAt = now.Add(time.Hour)
	}

	active, err := s.bitrixOAuthSessions.ActivateSession(ctx, pending.ID, bdomain.SessionActivateParams{
		PortalDomain: portal,
		RestEndpoint: restEndpoint,
		OAuthScopes:  oauthScopes,
		BitrixUserID: userID,
		UserName:     userName,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    expiresAt,
		UpdatedAt:    now,
	})
	if err != nil {
		return BitrixOAuthSessionView{}, err
	}

	s.auditBitrixOAuthConnect(ctx, active)

	return sessionView(active, "oauth"), nil
}

func (s *Service) GetBitrixOAuthSession(ctx context.Context, sessionID string) (BitrixOAuthSessionView, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return BitrixOAuthSessionView{Connected: false, AuthMode: "none"}, nil
	}
	session, err := s.ensureActiveBitrixSession(ctx, sessionID)
	if err != nil {
		return BitrixOAuthSessionView{}, err
	}
	return sessionView(session, "oauth"), nil
}

func (s *Service) RevokeBitrixOAuthSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("sessionId is required")
	}
	return s.bitrixOAuthSessions.RevokeSession(ctx, sessionID, s.now().UTC())
}

func (s *Service) ensureActiveBitrixSession(ctx context.Context, sessionID string) (bdomain.Session, error) {
	session, err := s.bitrixOAuthSessions.GetSessionByID(ctx, sessionID)
	if err != nil {
		return bdomain.Session{}, err
	}
	if session.Status != bdomain.StatusActive {
		return bdomain.Session{}, errors.New("сессия Bitrix не активна — войдите снова")
	}

	if session.ExpiresAt != nil && s.now().UTC().After(session.ExpiresAt.Add(-30*time.Second)) {
		if strings.TrimSpace(session.RefreshToken) == "" {
			return bdomain.Session{}, errors.New("сессия Bitrix истекла — войдите снова")
		}
		token, refreshErr := s.bitrixOAuth.RefreshToken(ctx, session.RefreshToken)
		if refreshErr != nil {
			return bdomain.Session{}, fmt.Errorf("обновление токена Bitrix: %w", refreshErr)
		}
		now := s.now().UTC()
		expiresAt := now.Add(time.Duration(token.ExpiresIn) * time.Second)
		session, err = s.bitrixOAuthSessions.UpdateSessionTokens(ctx, session.ID, bdomain.SessionTokenUpdateParams{
			AccessToken:  token.AccessToken,
			RefreshToken: fallbackString(token.RefreshToken, session.RefreshToken),
			RestEndpoint: strings.TrimSpace(token.ClientEndpoint),
			OAuthScopes:  strings.TrimSpace(token.Scope),
			ExpiresAt:    expiresAt,
			UpdatedAt:    now,
		})
		if err != nil {
			return bdomain.Session{}, err
		}
	}

	return session, nil
}

func (s *Service) tokenRESTForSession(ctx context.Context, sessionID string) (*bitrixclient.TokenREST, int, error) {
	session, err := s.ensureActiveBitrixSession(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}
	client := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
	if session.BitrixUserID <= 0 {
		return client, 0, errors.New("не удалось определить пользователя Bitrix для сессии")
	}
	return client, session.BitrixUserID, nil
}

func sessionView(session bdomain.Session, authMode string) BitrixOAuthSessionView {
	return BitrixOAuthSessionView{
		SessionID:        session.ID,
		Connected:        session.Status == bdomain.StatusActive,
		BitrixUserID:     session.BitrixUserID,
		UserName:         session.UserName,
		PortalDomain:     session.PortalDomain,
		OAuthScopes:      session.OAuthScopes,
		TaskScopeGranted: bitrixclient.HasTaskScope(session.OAuthScopes),
		AuthMode:         authMode,
	}
}

func BitrixSessionHeaderName() string {
	return bitrixSessionHeader
}
