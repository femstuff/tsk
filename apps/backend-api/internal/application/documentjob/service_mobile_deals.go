package documentjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tsk/backend-api/internal/infrastructure/cache"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

type BitrixMobileDealsResponse struct {
	Items    []bitrixclient.BitrixDealBrief `json:"items"`
	AuthMode string                         `json:"authMode,omitempty"`
}

func (s *Service) ListBitrixDealsForMobile(ctx context.Context, limit int, search string, oauthSessionID string, skipCache bool) (BitrixMobileDealsResponse, error) {
	var out BitrixMobileDealsResponse
	oauthSessionID = strings.TrimSpace(oauthSessionID)
	search = strings.TrimSpace(search)

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, sessionErr := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if sessionErr != nil {
			return out, sessionErr
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)

		cacheKey := cache.BitrixDealsKey(session.BitrixUserID, limit, search) + ":oauth:v3:" + oauthSessionID
		if !skipCache && s.cache != nil {
			if payload, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
				if json.Unmarshal(payload, &out) == nil {
					return out, nil
				}
			}
		}

		items, err := s.loadDealsViaOAuth(ctx, tokenClient, session.BitrixUserID, limit, search)
		if err != nil {
			return out, bitrixclient.DealsListUserError(err)
		}
		if len(items) == 0 && s.bitrix != nil && s.bitrix.WebhookConfigured() {
			if webhookItems, whErr := s.bitrix.ListDeals(ctx, limit, search); whErr == nil && len(webhookItems) > 0 {
				items = webhookItems
				out.AuthMode = "webhook"
			}
		} else {
			out.AuthMode = "oauth"
		}
		out.Items = items
		if s.cache != nil {
			if payload, err := json.Marshal(out); err == nil {
				_ = s.cache.Set(ctx, cacheKey, payload, cacheBitrixTasksTTL)
			}
		}
		return out, nil
	}

	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		return out, errors.New("Bitrix24 не настроен: войдите через OAuth или задайте BITRIX_WEBHOOK_URL")
	}

	cacheKey := cache.BitrixDealsKey(0, limit, search) + ":webhook:v3"
	if !skipCache && s.cache != nil {
		if payload, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
			if json.Unmarshal(payload, &out) == nil {
				return out, nil
			}
		}
	}

	items, err := s.bitrix.ListDeals(ctx, limit, search)
	if err != nil {
		return out, err
	}
	out.Items = items
	out.AuthMode = "webhook"
	if s.cache != nil {
		if payload, err := json.Marshal(out); err == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, cacheBitrixTasksTTL)
		}
	}
	return out, nil
}

func (s *Service) loadDealsViaOAuth(ctx context.Context, tokenClient *bitrixclient.TokenREST, userID int, limit int, search string) ([]bitrixclient.BitrixDealBrief, error) {
	items, err := tokenClient.ListDeals(ctx, limit, search)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}
	if userID > 0 {
		items, err = tokenClient.ListDealsForUser(ctx, userID, limit, search)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Service) GetBitrixDealForMobile(ctx context.Context, dealID string, oauthSessionID string) (bitrixclient.BitrixDealDetail, error) {
	oauthSessionID = strings.TrimSpace(oauthSessionID)
	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return bitrixclient.BitrixDealDetail{}, err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		detail, err := tokenClient.GetDealDetail(ctx, dealID)
		if err == nil {
			return detail, nil
		}
		if s.bitrix != nil && s.bitrix.WebhookConfigured() {
			return s.bitrix.GetDealDetail(ctx, dealID)
		}
		return bitrixclient.BitrixDealDetail{}, bitrixclient.DealsListUserError(err)
	}
	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		return bitrixclient.BitrixDealDetail{}, errors.New("Bitrix24 не настроен")
	}
	return s.bitrix.GetDealDetail(ctx, dealID)
}

func (s *Service) UpdateBitrixDealStageForMobile(ctx context.Context, dealID string, stageID string, oauthSessionID string) (bitrixclient.BitrixDealDetail, error) {
	dealID = strings.TrimSpace(dealID)
	stageID = strings.TrimSpace(stageID)
	if dealID == "" || stageID == "" {
		return bitrixclient.BitrixDealDetail{}, fmt.Errorf("deal id and stage id are required")
	}

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return bitrixclient.BitrixDealDetail{}, err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if err := tokenClient.UpdateDealStageByID(ctx, dealID, stageID); err == nil {
			return tokenClient.GetDealDetail(ctx, dealID)
		} else if s.bitrix != nil && s.bitrix.WebhookConfigured() {
			if whErr := s.bitrix.UpdateDealStageByID(ctx, dealID, stageID); whErr == nil {
				return s.bitrix.GetDealDetail(ctx, dealID)
			} else {
				return bitrixclient.BitrixDealDetail{}, whErr
			}
		}
		return bitrixclient.BitrixDealDetail{}, bitrixclient.DealsListUserError(err)
	}
	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		return bitrixclient.BitrixDealDetail{}, errors.New("Bitrix24 не настроен")
	}
	if err := s.bitrix.UpdateDealStageByID(ctx, dealID, stageID); err != nil {
		return bitrixclient.BitrixDealDetail{}, err
	}
	return s.bitrix.GetDealDetail(ctx, dealID)
}

func (s *Service) UpdateBitrixDealFieldsForMobile(ctx context.Context, dealID string, fields map[string]string, oauthSessionID string) (bitrixclient.BitrixDealDetail, error) {
	dealID = strings.TrimSpace(dealID)
	if dealID == "" {
		return bitrixclient.BitrixDealDetail{}, fmt.Errorf("deal id is required")
	}
	if len(fields) == 0 {
		return bitrixclient.BitrixDealDetail{}, fmt.Errorf("fields are required")
	}

	if oauthSessionID != "" && s.BitrixOAuthEnabled() {
		session, err := s.ensureActiveBitrixSession(ctx, oauthSessionID)
		if err != nil {
			return bitrixclient.BitrixDealDetail{}, err
		}
		tokenClient := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
		if err := tokenClient.UpdateDealFieldsByID(ctx, dealID, fields); err == nil {
			return tokenClient.GetDealDetail(ctx, dealID)
		} else if s.bitrix != nil && s.bitrix.WebhookConfigured() {
			if whErr := s.bitrix.UpdateDealFieldsByID(ctx, dealID, fields); whErr == nil {
				return s.bitrix.GetDealDetail(ctx, dealID)
			} else {
				return bitrixclient.BitrixDealDetail{}, whErr
			}
		}
		return bitrixclient.BitrixDealDetail{}, bitrixclient.DealsListUserError(err)
	}
	if s.bitrix == nil || !s.bitrix.WebhookConfigured() {
		return bitrixclient.BitrixDealDetail{}, errors.New("Bitrix24 не настроен")
	}
	if err := s.bitrix.UpdateDealFieldsByID(ctx, dealID, fields); err != nil {
		return bitrixclient.BitrixDealDetail{}, err
	}
	return s.bitrix.GetDealDetail(ctx, dealID)
}
