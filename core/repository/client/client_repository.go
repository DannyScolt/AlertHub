package client

import (
	"context"
	"errors"

	domain "alerthub/core/domain/client"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClientNotFound = errors.New("client not found")

type ClientRepository interface {
	Create(ctx context.Context, client domain.Client) (domain.Client, error)
	FindByEmail(ctx context.Context, email string) (domain.Client, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Client, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

type clientRepository struct{ db *pgxpool.Pool }

func NewClientRepository(db *pgxpool.Pool) ClientRepository { return &clientRepository{db: db} }

func (r *clientRepository) Create(ctx context.Context, c domain.Client) (domain.Client, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO clients (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, email, password_hash, name, created_at, updated_at`, c.Email, c.PasswordHash, c.Name)
	if err := row.Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return domain.Client{}, err
	}
	return c, nil
}

func (r *clientRepository) FindByEmail(ctx context.Context, email string) (domain.Client, error) {
	var c domain.Client
	err := r.db.QueryRow(ctx, `SELECT id, email, password_hash, name, created_at, updated_at FROM clients WHERE lower(email)=lower($1)`, email).Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Client{}, ErrClientNotFound
	}
	return c, err
}

func (r *clientRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	var c domain.Client
	err := r.db.QueryRow(ctx, `SELECT id, email, password_hash, name, created_at, updated_at FROM clients WHERE id=$1`, id).Scan(&c.ID, &c.Email, &c.PasswordHash, &c.Name, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Client{}, ErrClientNotFound
	}
	return c, err
}

func (r *clientRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clients WHERE lower(email)=lower($1))`, email).Scan(&exists)
	return exists, err
}
