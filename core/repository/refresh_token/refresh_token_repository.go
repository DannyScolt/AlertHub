package refresh_token

import (
	"context"
	"errors"
	"net"

	domain "alerthub/core/domain/refresh_token"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository interface {
	Create(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error)
	FindByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID, reason string) error
	RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error
	RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error
	RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error
	ListByClientID(ctx context.Context, clientID uuid.UUID) ([]domain.RefreshToken, error)
}

type refreshTokenRepository struct{ db *pgxpool.Pool }

func NewRefreshTokenRepository(db *pgxpool.Pool) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, t domain.RefreshToken) (domain.RefreshToken, error) {
	var ip *net.IP
	if t.IPAddress != nil {
		ip = t.IPAddress
	}
	err := r.db.QueryRow(ctx, `INSERT INTO refresh_tokens (client_id, token_hash, token_family, parent_id, expires_at, user_agent, ip_address) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, client_id, token_hash, token_family, parent_id, replaced_by_id, expires_at, created_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address`, t.ClientID, t.TokenHash, t.TokenFamily, t.ParentID, t.ExpiresAt, t.UserAgent, ip).Scan(&t.ID, &t.ClientID, &t.TokenHash, &t.TokenFamily, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress)
	return t, err
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.db.QueryRow(ctx, `SELECT id, client_id, token_hash, token_family, parent_id, replaced_by_id, expires_at, created_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address FROM refresh_tokens WHERE token_hash=$1`, tokenHash).Scan(&t.ID, &t.ClientID, &t.TokenHash, &t.TokenFamily, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, ErrRefreshTokenNotFound
	}
	return t, err
}

func (r *refreshTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET last_used_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *refreshTokenRepository) SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET replaced_by_id=$2 WHERE id=$1`, id, replacedByID)
	return err
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2) WHERE id=$1`, id, reason)
	return err
}

func (r *refreshTokenRepository) RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error {
	cmd, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $3) WHERE client_id=$1 AND id=$2`, clientID, sessionID, reason)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *refreshTokenRepository) RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2) WHERE token_family=$1 AND revoked_at IS NULL`, tokenFamily, reason)
	return err
}

func (r *refreshTokenRepository) RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2) WHERE client_id=$1 AND revoked_at IS NULL`, clientID, reason)
	return err
}

func (r *refreshTokenRepository) ListByClientID(ctx context.Context, clientID uuid.UUID) ([]domain.RefreshToken, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, token_hash, token_family, parent_id, replaced_by_id, expires_at, created_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address FROM refresh_tokens WHERE client_id=$1 ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]domain.RefreshToken, 0)
	for rows.Next() {
		var t domain.RefreshToken
		if err := rows.Scan(&t.ID, &t.ClientID, &t.TokenHash, &t.TokenFamily, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
