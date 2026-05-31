package documentjob

import (
	"context"
	"errors"
	"strings"

	"tsk/backend-api/internal/integrations/bitrixclient"
)

type BitrixMobileNotificationsResponse struct {
	Items    []bitrixclient.BitrixNotification `json:"items"`
	AuthMode string                            `json:"authMode,omitempty"`
}

func (s *Service) ListBitrixNotificationsForMobile(ctx context.Context, limit int, oauthSessionID string) (BitrixMobileNotificationsResponse, error) {
	var out BitrixMobileNotificationsResponse
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return out, err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		items, err := tokenClient.ListNotifications(ctx, limit)
		if err != nil {
			return out, bitrixclient.NotifyListUserError(err)
		}
		out.Items = items
		out.AuthMode = "oauth"
		return out, nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return out, errors.New("войдите в Bitrix24 в приложении, чтобы видеть уведомления")
		}
		return out, errors.New("Bitrix24 не настроен")
	}

	items, err := s.bitrix.ListNotifications(ctx, limit)
	if err != nil {
		return out, err
	}
	out.Items = items
	out.AuthMode = "webhook"
	return out, nil
}

func (s *Service) MarkBitrixNotificationReadForMobile(ctx context.Context, notificationID, oauthSessionID string) error {
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return errors.New("notification id is required")
	}
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if err := tokenClient.MarkNotificationRead(ctx, notificationID); err != nil {
			return bitrixclient.NotifyListUserError(err)
		}
		return nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return errors.New("войдите в Bitrix24 в приложении, чтобы отмечать уведомления")
		}
		return errors.New("Bitrix24 не настроен")
	}

	return s.bitrix.MarkNotificationRead(ctx, notificationID)
}

func (s *Service) MarkAllBitrixNotificationsReadForMobile(ctx context.Context, oauthSessionID string) error {
	oauthSessionID = strings.TrimSpace(oauthSessionID)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if err := tokenClient.MarkAllNotificationsRead(ctx); err != nil {
			return bitrixclient.NotifyListUserError(err)
		}
		return nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		if s.BitrixOAuthEnabled() {
			return errors.New("войдите в Bitrix24 в приложении, чтобы отмечать уведомления")
		}
		return errors.New("Bitrix24 не настроен")
	}

	return s.bitrix.MarkAllNotificationsRead(ctx)
}
