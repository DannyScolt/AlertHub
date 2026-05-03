package client_token

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	domain "alerthub/core/domain/client_token"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClientTokenNotFound = errors.New("client token not found")

type ClientTokenRepository interface {
	Create(ctx context.Context, token domain.ClientToken) (domain.ClientToken, error)
	FindByHash(ctx context.Context, tokenHash string) (domain.ClientToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID, reason string) error
	RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error
	RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error
	RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error
	ListByClientID(ctx context.Context, clientID uuid.UUID) ([]domain.ClientToken, error)
}

type clientTokenRepository struct{ db *pgxpool.Pool }

func NewClientTokenRepository(db *pgxpool.Pool) ClientTokenRepository {
	return &clientTokenRepository{db: db}
}

func (r *clientTokenRepository) Create(ctx context.Context, t domain.ClientToken) (domain.ClientToken, error) {
	var ip *net.IP
	if t.IPAddress != nil {
		ip = t.IPAddress
	}
	abilities := t.Abilities
	if abilities == nil {
		abilities = []string{}
	}
	abilitiesJSON, err := json.Marshal(abilities)
	if err != nil {
		return domain.ClientToken{}, err
	}
	err = r.db.QueryRow(ctx, `INSERT INTO client_tokens (client_id, name, token_hash, token_family, abilities, parent_id, expires_at, user_agent, ip_address) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, client_id, name, token_hash, token_family, abilities, parent_id, replaced_by_id, expires_at, created_at, updated_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address`, t.ClientID, tokenName(t.Name), t.TokenHash, t.TokenFamily, abilitiesJSON, t.ParentID, t.ExpiresAt, t.UserAgent, ip).Scan(&t.ID, &t.ClientID, &t.Name, &t.TokenHash, &t.TokenFamily, &abilitiesJSON, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress)
	if err != nil {
		return t, err
	}
	t.Abilities, err = decodeAbilities(abilitiesJSON)
	return t, err
}

func (r *clientTokenRepository) FindByHash(ctx context.Context, tokenHash string) (domain.ClientToken, error) {
	var t domain.ClientToken
	var abilitiesJSON []byte
	err := r.db.QueryRow(ctx, `SELECT id, client_id, name, token_hash, token_family, abilities, parent_id, replaced_by_id, expires_at, created_at, updated_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address FROM client_tokens WHERE token_hash=$1`, tokenHash).Scan(&t.ID, &t.ClientID, &t.Name, &t.TokenHash, &t.TokenFamily, &abilitiesJSON, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ClientToken{}, ErrClientTokenNotFound
	}
	if err != nil {
		return t, err
	}
	t.Abilities, err = decodeAbilities(abilitiesJSON)
	return t, err
}

func (r *clientTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE client_tokens SET last_used_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *clientTokenRepository) SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE client_tokens SET replaced_by_id=$2, updated_at=NOW() WHERE id=$1`, id, replacedByID)
	return err
}

func (r *clientTokenRepository) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE client_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2), updated_at=NOW() WHERE id=$1`, id, reason)
	return err
}

func (r *clientTokenRepository) RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error {
	cmd, err := r.db.Exec(ctx, `UPDATE client_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $3), updated_at=NOW() WHERE client_id=$1 AND id=$2`, clientID, sessionID, reason)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrClientTokenNotFound
	}
	return nil
}

func (r *clientTokenRepository) RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE client_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2), updated_at=NOW() WHERE token_family=$1 AND revoked_at IS NULL`, tokenFamily, reason)
	return err
}

func (r *clientTokenRepository) RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `UPDATE client_tokens SET revoked_at=COALESCE(revoked_at, NOW()), revoke_reason=COALESCE(revoke_reason, $2), updated_at=NOW() WHERE client_id=$1 AND revoked_at IS NULL`, clientID, reason)
	return err
}

func (r *clientTokenRepository) ListByClientID(ctx context.Context, clientID uuid.UUID) ([]domain.ClientToken, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, name, token_hash, token_family, abilities, parent_id, replaced_by_id, expires_at, created_at, updated_at, last_used_at, revoked_at, revoke_reason, user_agent, ip_address FROM client_tokens WHERE client_id=$1 ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]domain.ClientToken, 0)
	for rows.Next() {
		var t domain.ClientToken
		var abilitiesJSON []byte
		if err := rows.Scan(&t.ID, &t.ClientID, &t.Name, &t.TokenHash, &t.TokenFamily, &abilitiesJSON, &t.ParentID, &t.ReplacedByID, &t.ExpiresAt, &t.CreatedAt, &t.UpdatedAt, &t.LastUsedAt, &t.RevokedAt, &t.RevokeReason, &t.UserAgent, &t.IPAddress); err != nil {
			return nil, err
		}
		t.Abilities, err = decodeAbilities(abilitiesJSON)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func tokenName(name string) string {
	if name == "" {
		return "auth_session"
	}
	return name
}

func decodeAbilities(data []byte) ([]string, error) {
	var abilities []string
	if len(data) == 0 {
		return []string{}, nil
	}
	if err := json.Unmarshal(data, &abilities); err != nil {
		return nil, err
	}
	return abilities, nil
}
