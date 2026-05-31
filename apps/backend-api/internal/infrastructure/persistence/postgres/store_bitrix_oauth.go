package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	domain "tsk/backend-api/internal/domain/bitrixoauth"
)

func scanBitrixOAuthSession(row pgx.Row) (domain.Session, error) {
	var session domain.Session
	err := row.Scan(
		&session.ID,
		&session.State,
		&session.Status,
		&session.PortalDomain,
		&session.RestEndpoint,
		&session.OAuthScopes,
		&session.BitrixUserID,
		&session.UserName,
		&session.AccessToken,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	return session, err
}

const bitrixOAuthSessionColumns = `
	id, state, status, portal_domain, rest_endpoint, oauth_scopes, bitrix_user_id, user_name,
	access_token, refresh_token, expires_at, created_at, updated_at
`

func (s *Store) CreateSession(ctx context.Context, params domain.SessionCreateParams) (domain.Session, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO bitrix_oauth_sessions (
			id, state, status, portal_domain, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+bitrixOAuthSessionColumns+`
	`, params.ID, params.State, params.Status, params.PortalDomain, params.CreatedAt, params.UpdatedAt)

	session, err := scanBitrixOAuthSession(row)
	if err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) GetSessionByID(ctx context.Context, id string) (domain.Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+bitrixOAuthSessionColumns+`
		FROM bitrix_oauth_sessions
		WHERE id = $1
	`, id)

	session, err := scanBitrixOAuthSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) GetSessionByState(ctx context.Context, state string) (domain.Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+bitrixOAuthSessionColumns+`
		FROM bitrix_oauth_sessions
		WHERE state = $1
	`, state)

	session, err := scanBitrixOAuthSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) ActivateSession(ctx context.Context, id string, params domain.SessionActivateParams) (domain.Session, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE bitrix_oauth_sessions
		SET status = $2,
		    portal_domain = $3,
		    rest_endpoint = $4,
		    oauth_scopes = $5,
		    bitrix_user_id = $6,
		    user_name = $7,
		    access_token = $8,
		    refresh_token = $9,
		    expires_at = $10,
		    updated_at = $11
		WHERE id = $1
		RETURNING `+bitrixOAuthSessionColumns+`
	`, id, domain.StatusActive, params.PortalDomain, params.RestEndpoint, params.OAuthScopes,
		params.BitrixUserID, params.UserName, params.AccessToken, params.RefreshToken,
		params.ExpiresAt, params.UpdatedAt)

	session, err := scanBitrixOAuthSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) UpdateSessionTokens(ctx context.Context, id string, params domain.SessionTokenUpdateParams) (domain.Session, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE bitrix_oauth_sessions
		SET access_token = $2,
		    refresh_token = $3,
		    rest_endpoint = COALESCE(NULLIF($4, ''), rest_endpoint),
		    oauth_scopes = COALESCE(NULLIF($5, ''), oauth_scopes),
		    expires_at = $6,
		    updated_at = $7
		WHERE id = $1 AND status = $8
		RETURNING `+bitrixOAuthSessionColumns+`
	`, id, params.AccessToken, params.RefreshToken, params.RestEndpoint, params.OAuthScopes,
		params.ExpiresAt, params.UpdatedAt, domain.StatusActive)

	session, err := scanBitrixOAuthSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) RevokeSession(ctx context.Context, id string, updatedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE bitrix_oauth_sessions
		SET status = $2, updated_at = $3
		WHERE id = $1
	`, id, domain.StatusRevoked, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]domain.Session, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+bitrixOAuthSessionColumns+`
		FROM bitrix_oauth_sessions
		WHERE status IN ($1, $2)
		ORDER BY updated_at DESC
		LIMIT $3
	`, domain.StatusActive, domain.StatusRevoked, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Session, 0)
	for rows.Next() {
		session, err := scanBitrixOAuthSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}
