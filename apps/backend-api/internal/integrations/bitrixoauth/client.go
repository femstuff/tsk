package bitrixoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const tokenEndpoint = "https://oauth.bitrix.info/oauth/token/"

type Config struct {
	ClientID     string
	ClientSecret string
	PortalDomain string
	RedirectURI  string
	MobileScheme string
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		strings.TrimSpace(c.PortalDomain) != "" &&
		strings.TrimSpace(c.RedirectURI) != ""
}

type TokenResponse struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	ExpiresIn      int    `json:"expires_in"`
	Domain         string `json:"domain"`
	MemberID       string `json:"member_id"`
	UserID         int    `json:"user_id"`
	ClientEndpoint string `json:"client_endpoint"`
	ServerEndpoint string `json:"server_endpoint"`
	Scope          string `json:"scope"`
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, httpClient: httpClient}
}

func (c *Client) AuthorizeURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.cfg.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", c.cfg.RedirectURI)
	params.Set("state", state)
	return fmt.Sprintf("https://%s/oauth/authorize/?%s", c.cfg.PortalDomain, params.Encode())
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (TokenResponse, error) {
	return c.postToken(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"client_id":    {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"code":         {code},
		"redirect_uri": {c.cfg.RedirectURI},
	})
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (TokenResponse, error) {
	return c.postToken(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"refresh_token": {refreshToken},
	})
}

func (c *Client) postToken(ctx context.Context, form url.Values) (TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return TokenResponse{}, fmt.Errorf("bitrix oauth token HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed TokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return TokenResponse{}, fmt.Errorf("bitrix oauth token JSON: %w", err)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return TokenResponse{}, fmt.Errorf("bitrix oauth: empty access_token")
	}
	return parsed, nil
}

func (c *Client) CurrentUser(ctx context.Context, portalDomain, restEndpoint, accessToken string) (userID int, userName string, err error) {
	base := restEndpoint
	if base == "" {
		host := strings.TrimSpace(portalDomain)
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimSuffix(host, "/")
		if host == "" {
			return 0, "", fmt.Errorf("bitrix portal domain is required")
		}
		base = "https://" + host + "/rest/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	endpoint := base + "user.current?auth=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	if resp.StatusCode >= 300 {
		return 0, "", fmt.Errorf("user.current HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Result struct {
			ID        string `json:"ID"`
			NAME      string `json:"NAME"`
			LAST_NAME string `json:"LAST_NAME"`
		} `json:"result"`
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, "", err
	}
	if envelope.Error != nil || envelope.ErrorDescription != "" {
		return 0, "", fmt.Errorf("user.current error: %v %s", envelope.Error, envelope.ErrorDescription)
	}

	id := 0
	if envelope.Result.ID != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(envelope.Result.ID)); err == nil {
			id = parsed
		}
	}
	name := strings.TrimSpace(strings.TrimSpace(envelope.Result.NAME + " " + envelope.Result.LAST_NAME))
	return id, name, nil
}

func PortalDomainFromWebhook(webhookURL string) string {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return ""
	}
	u, err := url.Parse(webhookURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func MobileRedirectURL(scheme, sessionID string) string {
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		scheme = "tsk"
	}
	scheme = strings.TrimSuffix(scheme, "://")
	return fmt.Sprintf("%s://bitrix-oauth?sessionId=%s&connected=1", scheme, url.QueryEscape(sessionID))
}
