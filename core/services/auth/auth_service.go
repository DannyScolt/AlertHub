package auth

import (
	"context"
	"errors"
	"net"
	"time"

	"alerthub/core/config"
	clientDomain "alerthub/core/domain/client"
	clientTokenDomain "alerthub/core/domain/client_token"
	authDto "alerthub/core/dto/auth"
	clientRepo "alerthub/core/repository/client"
	clientTokenRepo "alerthub/core/repository/client_token"
	"alerthub/core/utils/apikey"
	"alerthub/core/utils/password"
	"alerthub/core/utils/token"

	"github.com/google/uuid"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type AuthService interface {
	Register(ctx context.Context, req authDto.RegisterRequest) (authDto.AuthData, error)
	Login(ctx context.Context, req authDto.LoginRequest, userAgent string, ip net.IP) (authDto.AuthData, error)
	Refresh(ctx context.Context, rawRefreshToken string, userAgent string, ip net.IP) (authDto.AuthData, error)
	Logout(ctx context.Context, clientID uuid.UUID) error
	LogoutAll(ctx context.Context, clientID uuid.UUID) error
	ListSessions(ctx context.Context, clientID uuid.UUID) ([]authDto.SessionResponse, error)
	RevokeSession(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID) error
}

type authService struct {
	cfg          *config.Config
	clients      clientRepo.ClientRepository
	clientTokens clientTokenRepo.ClientTokenRepository
}

func NewAuthService(cfg *config.Config, clients clientRepo.ClientRepository, clientTokens clientTokenRepo.ClientTokenRepository) AuthService {
	return &authService{cfg: cfg, clients: clients, clientTokens: clientTokens}
}

func (s *authService) Register(ctx context.Context, req authDto.RegisterRequest) (authDto.AuthData, error) {
	exists, err := s.clients.EmailExists(ctx, req.Email)
	if err != nil {
		return authDto.AuthData{}, err
	}
	if exists {
		return authDto.AuthData{}, ErrEmailAlreadyExists
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return authDto.AuthData{}, err
	}

	created, err := s.clients.Create(ctx, clientDomain.Client{Email: req.Email, Name: req.Name, PasswordHash: hashedPassword})
	if err != nil {
		return authDto.AuthData{}, err
	}

	return s.issueTokens(ctx, created, nil, nil, uuid.Nil)
}

func (s *authService) Login(ctx context.Context, req authDto.LoginRequest, userAgent string, ip net.IP) (authDto.AuthData, error) {
	client, err := s.clients.FindByEmail(ctx, req.Email)
	if err != nil {
		return authDto.AuthData{}, ErrInvalidCredentials
	}
	if !password.Verify(client.PasswordHash, req.Password) {
		return authDto.AuthData{}, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, client, &userAgent, &ip, uuid.Nil)
}

func (s *authService) Refresh(ctx context.Context, rawRefreshToken string, userAgent string, ip net.IP) (authDto.AuthData, error) {
	hash := apikey.Hash(rawRefreshToken)
	stored, err := s.clientTokens.FindByHash(ctx, hash)
	if err != nil {
		return authDto.AuthData{}, ErrInvalidRefreshToken
	}
	now := time.Now()
	if stored.IsExpired(now) || stored.IsRevoked() {
		return authDto.AuthData{}, ErrInvalidRefreshToken
	}
	if stored.IsReplaced() {
		_ = s.clientTokens.RevokeFamily(ctx, stored.TokenFamily, "refresh_token_reuse")
		return authDto.AuthData{}, ErrInvalidRefreshToken
	}

	client, err := s.clients.FindByID(ctx, stored.ClientID)
	if err != nil {
		return authDto.AuthData{}, err
	}
	if err := s.clientTokens.MarkUsed(ctx, stored.ID); err != nil {
		return authDto.AuthData{}, err
	}

	data, err := s.issueTokens(ctx, client, &userAgent, &ip, stored.TokenFamily)
	if err != nil {
		return authDto.AuthData{}, err
	}
	newStored, err := s.clientTokens.FindByHash(ctx, apikey.Hash(data.RefreshToken))
	if err != nil {
		return authDto.AuthData{}, err
	}
	if err := s.clientTokens.SetReplacedBy(ctx, stored.ID, newStored.ID); err != nil {
		return authDto.AuthData{}, err
	}
	return data, nil
}

func (s *authService) Logout(ctx context.Context, clientID uuid.UUID) error {
	return s.clientTokens.RevokeAllByClientID(ctx, clientID, "logout")
}

func (s *authService) LogoutAll(ctx context.Context, clientID uuid.UUID) error {
	return s.clientTokens.RevokeAllByClientID(ctx, clientID, "logout_all")
}

func (s *authService) ListSessions(ctx context.Context, clientID uuid.UUID) ([]authDto.SessionResponse, error) {
	tokens, err := s.clientTokens.ListByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	result := make([]authDto.SessionResponse, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, toSessionResponse(t))
	}
	return result, nil
}

func (s *authService) RevokeSession(ctx context.Context, clientID uuid.UUID, sessionID uuid.UUID) error {
	if err := s.clientTokens.RevokeByClientID(ctx, clientID, sessionID, "session_revoked"); err != nil {
		return ErrInvalidRefreshToken
	}
	return nil
}

func (s *authService) issueTokens(ctx context.Context, c clientDomain.Client, userAgent *string, ip *net.IP, tokenFamily uuid.UUID) (authDto.AuthData, error) {
	accessToken, err := token.GenerateAccessToken(c.ID, s.cfg.JWTAccessSecret, s.cfg.JWTAccessTTL)
	if err != nil {
		return authDto.AuthData{}, err
	}
	rawRefreshToken, err := apikey.Generate("ah_refresh")
	if err != nil {
		return authDto.AuthData{}, err
	}
	if tokenFamily == uuid.Nil {
		tokenFamily = uuid.New()
	}
	_, err = s.clientTokens.Create(ctx, clientTokenDomain.ClientToken{ClientID: c.ID, Name: "auth_session", TokenHash: apikey.Hash(rawRefreshToken), TokenFamily: tokenFamily, Abilities: []string{}, ExpiresAt: time.Now().Add(s.cfg.JWTRefreshTTL), UserAgent: userAgent, IPAddress: ip})
	if err != nil {
		return authDto.AuthData{}, err
	}
	return authDto.AuthData{AccessToken: accessToken, RefreshToken: rawRefreshToken, TokenType: "Bearer", ExpiresIn: int64(s.cfg.JWTAccessTTL.Seconds())}, nil
}

func toSessionResponse(t clientTokenDomain.ClientToken) authDto.SessionResponse {
	var ip *string
	if t.IPAddress != nil {
		value := t.IPAddress.String()
		ip = &value
	}
	return authDto.SessionResponse{ID: t.ID.String(), TokenFamily: t.TokenFamily.String(), ExpiresAt: t.ExpiresAt, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt, RevokedAt: t.RevokedAt, RevokeReason: t.RevokeReason, UserAgent: t.UserAgent, IPAddress: ip}
}
