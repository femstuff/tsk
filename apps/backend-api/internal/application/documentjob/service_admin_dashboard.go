package documentjob

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	bdomain "tsk/backend-api/internal/domain/bitrixoauth"
	"tsk/backend-api/internal/integrations/bitrixclient"
)

type AdminBitrixUserView struct {
	SessionID    string     `json:"sessionId"`
	BitrixUserID int        `json:"bitrixUserId"`
	UserName     string     `json:"userName"`
	PortalDomain string     `json:"portalDomain"`
	OAuthScopes  string     `json:"oauthScopes,omitempty"`
	TaskCount    int        `json:"taskCount"`
	Status       string     `json:"status"`
	ConnectedAt  time.Time  `json:"connectedAt"`
	LastActiveAt time.Time  `json:"lastActiveAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
}

type AdminAuthActivityView struct {
	BitrixUserID int       `json:"bitrixUserId"`
	UserName     string    `json:"userName"`
	PortalDomain string    `json:"portalDomain"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurredAt"`
	Message      string    `json:"message"`
}

type AdminBitrixTaskStatsView struct {
	Total      int `json:"total"`
	TotalOpen  int `json:"totalOpen"`
	InProgress int `json:"inProgress"`
	Overdue    int `json:"overdue"`
}

type AdminBitrixTaskItemView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	StatusLabel string `json:"statusLabel"`
	Deadline    string `json:"deadline,omitempty"`
	ClosedDate  string `json:"closedDate,omitempty"`
}

type AdminDashboardView struct {
	BitrixTasks        AdminBitrixTaskStatsView  `json:"bitrixTasks"`
	BitrixTaskItems    []AdminBitrixTaskItemView `json:"bitrixTaskItems"`
	AuthorizedUsers    int                       `json:"authorizedUsers"`
	VoiceActivityToday int                      `json:"voiceActivityToday"`
	VoiceActivityWeek  int                      `json:"voiceActivityWeek"`
	Users              []AdminBitrixUserView    `json:"users"`
	RecentAuth         []AdminAuthActivityView  `json:"recentAuth"`
}

func (s *Service) AdminDashboard(ctx context.Context) (AdminDashboardView, error) {
	var out AdminDashboardView
	now := s.now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekAgo := now.Add(-7 * 24 * time.Hour)

	if s.events != nil {
		if count, err := s.events.CountVoiceEventsSince(ctx, startOfDay); err == nil {
			out.VoiceActivityToday = count
		}
		if count, err := s.events.CountVoiceEventsSince(ctx, weekAgo); err == nil {
			out.VoiceActivityWeek = count
		}
	}

	sessions, err := s.bitrixOAuthSessions.ListSessions(ctx, 100)
	if err != nil {
		return out, err
	}

	activeSessions := make([]AdminBitrixUserView, 0)
	recentAuth := make([]AdminAuthActivityView, 0, len(sessions))
	allTasks := make(map[string]bitrixclient.BitrixTaskBrief)

	for _, session := range sessions {
		if session.Status == bdomain.StatusActive {
			out.AuthorizedUsers++
		}

		taskCount := 0
		if session.Status == bdomain.StatusActive && session.BitrixUserID > 0 {
			tasks, err := s.adminBitrixTasksForSession(ctx, session)
			if err == nil {
				taskCount = len(tasks)
				for _, task := range tasks {
					allTasks[task.ID] = task
				}
			}
		}

		userView := AdminBitrixUserView{
			SessionID:    session.ID,
			BitrixUserID: session.BitrixUserID,
			UserName:     session.UserName,
			PortalDomain: session.PortalDomain,
			OAuthScopes:  session.OAuthScopes,
			TaskCount:    taskCount,
			Status:       session.Status,
			ConnectedAt:  session.CreatedAt,
			LastActiveAt: session.UpdatedAt,
			ExpiresAt:    session.ExpiresAt,
		}
		if session.Status == bdomain.StatusActive {
			activeSessions = append(activeSessions, userView)
		}

		name := strings.TrimSpace(session.UserName)
		if name == "" && session.BitrixUserID > 0 {
			name = fmt.Sprintf("id %d", session.BitrixUserID)
		}
		msg := "Вход в Bitrix24"
		if session.Status == bdomain.StatusRevoked {
			msg = "Выход из Bitrix24"
		}
		recentAuth = append(recentAuth, AdminAuthActivityView{
			BitrixUserID: session.BitrixUserID,
			UserName:     name,
			PortalDomain: session.PortalDomain,
			Status:       session.Status,
			OccurredAt:   session.UpdatedAt,
			Message:      msg,
		})
	}

	if len(allTasks) == 0 && s.bitrix != nil && s.bitrix.WebhookConfigured() {
		if tasks, err := s.bitrix.ListTasks(ctx, 100); err == nil {
			for _, task := range tasks {
				allTasks[task.ID] = task
			}
		}
	}

	taskItems := make([]bitrixclient.BitrixTaskBrief, 0, len(allTasks))
	for _, task := range allTasks {
		taskItems = append(taskItems, task)
	}
	sort.Slice(taskItems, func(i, j int) bool { return taskItems[i].ID > taskItems[j].ID })

	stats := computeBitrixTaskStats(taskItems, now.In(time.Local))
	out.BitrixTasks = AdminBitrixTaskStatsView{
		Total:      len(taskItems),
		TotalOpen:  stats.TotalOpen,
		InProgress: stats.InProgress,
		Overdue:    stats.Overdue,
	}
	out.BitrixTaskItems = mapAdminBitrixTaskItems(taskItems)
	out.Users = activeSessions
	if len(recentAuth) > 8 {
		recentAuth = recentAuth[:8]
	}
	out.RecentAuth = recentAuth

	return out, nil
}

func (s *Service) adminBitrixTasksForSession(ctx context.Context, session bdomain.Session) ([]bitrixclient.BitrixTaskBrief, error) {
	if session.BitrixUserID <= 0 {
		return nil, fmt.Errorf("bitrix user id is required")
	}
	if strings.TrimSpace(session.AccessToken) == "" {
		return nil, fmt.Errorf("oauth session has no token")
	}
	client := bitrixclient.NewTokenREST(session.PortalDomain, session.RestEndpoint, session.AccessToken, s.httpClient)
	return client.ListTasksForUser(ctx, session.BitrixUserID, 100)
}

func (s *Service) auditBitrixOAuthConnect(ctx context.Context, session bdomain.Session) {
	if s.events == nil {
		return
	}
	name := strings.TrimSpace(session.UserName)
	if name == "" && session.BitrixUserID > 0 {
		name = fmt.Sprintf("id %d", session.BitrixUserID)
	}
	details, _ := json.Marshal(map[string]any{
		"sessionId":    session.ID,
		"bitrixUserId": session.BitrixUserID,
		"userName":     session.UserName,
		"portalDomain": session.PortalDomain,
		"oauthScopes":  session.OAuthScopes,
	})
	_, _ = s.createEvent(ctx, nil, "info", "bitrix.oauth.connected",
		"Авторизация Bitrix24: "+name, string(details))
}
