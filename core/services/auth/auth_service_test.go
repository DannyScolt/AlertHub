package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"alerthub/core/config"
	clientDomain "alerthub/core/domain/client"
	refreshDomain "alerthub/core/domain/refresh_token"
	authDto "alerthub/core/dto/auth"
	clientRepo "alerthub/core/repository/client"
	refreshRepo "alerthub/core/repository/refresh_token"

	"github.com/google/uuid"
)

type authClientRepoStub struct{}

func (r *authClientRepoStub) Create(ctx context.Context, client clientDomain.Client) (clientDomain.Client, error) {
	return clientDomain.Client{}, nil
}

func (r *authClientRepoStub) FindByEmail(ctx context.Context, email string) (clientDomain.Client, error) {
	return clientDomain.Client{}, clientRepo.ErrClientNotFound
}

func (r *authClientRepoStub) FindByID(ctx context.Context, id uuid.UUID) (clientDomain.Client, error) {
	return clientDomain.Client{}, clientRepo.ErrClientNotFound
}

func (r *authClientRepoStub) EmailExists(ctx context.Context, email string) (bool, error) {
	return false, nil
}

type authRefreshRepoStub struct {
	revokedClientID  uuid.UUID
	revokedSessionID uuid.UUID
	revokedReason    string
}

func (r *authRefreshRepoStub) Create(ctx context.Context, token refreshDomain.RefreshToken) (refreshDomain.RefreshToken, error) {
	return refreshDomain.RefreshToken{}, nil
}

func (r *authRefreshRepoStub) FindByHash(ctx context.Context, tokenHash string) (refreshDomain.RefreshToken, error) {
	return refreshDomain.RefreshToken{}, refreshRepo.ErrRefreshTokenNotFound
}

func (r *authRefreshRepoStub) MarkUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (r *authRefreshRepoStub) SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error {
	return nil
}

func (r *authRefreshRepoStub) Revoke(ctx context.Context, id uuid.UUID, reason string) error {
	return nil
}

func (r *authRefreshRepoStub) RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error {
	r.revokedClientID = clientID
	r.revokedSessionID = sessionID
	r.revokedReason = reason
	return nil
}

func (r *authRefreshRepoStub) RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error {
	return nil
}

func (r *authRefreshRepoStub) RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error {
	return nil
}

func (r *authRefreshRepoStub) ListByClientID(ctx context.Context, clientID uuid.UUID) ([]refreshDomain.RefreshToken, error) {
	return nil, errors.New("ListByClientID should not be called when revoking one session")
}

func TestRevokeSessionRevokesDirectlyByClientOwnership(t *testing.T) {
	clientID := uuid.New()
	sessionID := uuid.New()
	repo := &authRefreshRepoStub{}
	service := NewAuthService(&config.Config{JWTAccessTTL: 15 * time.Minute, JWTRefreshTTL: 30 * 24 * time.Hour}, &authClientRepoStub{}, repo)

	err := service.RevokeSession(context.Background(), clientID, sessionID)
	if err != nil {
		t.Fatalf("RevokeSession returned error: %v", err)
	}
	if repo.revokedClientID != clientID {
		t.Fatalf("expected client ID %s, got %s", clientID, repo.revokedClientID)
	}
	if repo.revokedSessionID != sessionID {
		t.Fatalf("expected session ID %s, got %s", sessionID, repo.revokedSessionID)
	}
	if repo.revokedReason != "session_revoked" {
		t.Fatalf("expected revoke reason session_revoked, got %s", repo.revokedReason)
	}
}

func TestAuthDataJSONContainsOnlyTokenFields(t *testing.T) {
	data := authDto.AuthData{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if _, ok := fields["client"]; ok {
		t.Fatalf("auth response should not include client field: %s", string(payload))
	}
	expectedFields := []string{"access_token", "refresh_token", "token_type", "expires_in"}
	for _, field := range expectedFields {
		if _, ok := fields[field]; !ok {
			t.Fatalf("expected auth response to include %s: %s", field, string(payload))
		}
	}
	if len(fields) != len(expectedFields) {
		t.Fatalf("expected only token fields, got %s", string(payload))
	}
}

var _ RefreshTokenRepoCompileCheck = (*authRefreshRepoStub)(nil)

type RefreshTokenRepoCompileCheck interface {
	Create(ctx context.Context, token refreshDomain.RefreshToken) (refreshDomain.RefreshToken, error)
	FindByHash(ctx context.Context, tokenHash string) (refreshDomain.RefreshToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	SetReplacedBy(ctx context.Context, id uuid.UUID, replacedByID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID, reason string) error
	RevokeByClientID(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID, reason string) error
	RevokeFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error
	RevokeAllByClientID(ctx context.Context, clientID uuid.UUID, reason string) error
	ListByClientID(ctx context.Context, clientID uuid.UUID) ([]refreshDomain.RefreshToken, error)
}

var _ = net.IP{}
