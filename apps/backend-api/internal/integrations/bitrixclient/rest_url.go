package bitrixclient

import (
	"net/url"
	"strings"
)

// NormalizePortalHost убирает схему и путь из домена портала Bitrix24.
func NormalizePortalHost(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimSuffix(raw, "/")
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

// PortalHostFromClientEndpoint извлекает домен портала из client_endpoint токена.
func PortalHostFromClientEndpoint(clientEndpoint string) string {
	ep := strings.TrimSpace(clientEndpoint)
	if ep == "" {
		return ""
	}
	if u, err := url.Parse(ep); err == nil && u.Host != "" {
		return u.Host
	}
	return NormalizePortalHost(ep)
}

// ResolvePortalHost выбирает домен портала, игнорируя oauth.bitrix.info.
func ResolvePortalHost(callbackDomain, configDomain, clientEndpoint, tokenDomain string) string {
	for _, candidate := range []string{
		callbackDomain,
		PortalHostFromClientEndpoint(clientEndpoint),
		configDomain,
		tokenDomain,
	} {
		host := NormalizePortalHost(candidate)
		if host != "" && host != "oauth.bitrix.info" {
			return host
		}
	}
	return NormalizePortalHost(configDomain)
}

// RestBaseURL — базовый URL REST API для OAuth-токена (client_endpoint или https://{portal}/rest/).
func RestBaseURL(portalDomain, clientEndpoint string) string {
	if ep := strings.TrimSpace(clientEndpoint); ep != "" {
		ep = strings.TrimSuffix(ep, "/")
		return ep + "/"
	}
	host := NormalizePortalHost(portalDomain)
	if host == "" {
		return ""
	}
	return "https://" + host + "/rest/"
}

// RestAPIV3Base вставляет сегмент /api/ в REST URL для методов tasks v3 (например chat.message.send).
func RestAPIV3Base(restBase string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(restBase), "/")
	if strings.Contains(trimmed, "/rest/api") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/rest") {
		return trimmed + "/api"
	}
	const marker = "/rest/"
	idx := strings.Index(trimmed, marker)
	if idx < 0 {
		return trimmed
	}
	return trimmed[:idx+len(marker)] + "api/" + trimmed[idx+len(marker):]
}
