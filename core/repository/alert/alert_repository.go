package alert

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "alerthub/core/domain/alert"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlertNotFound       = errors.New("alert not found")
	ErrConstraintViolation = errors.New("constraint violation")
)

type AlertRepository interface {
	Create(ctx context.Context, alert domain.Alert) (domain.Alert, error)
	CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error)
	FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error)
	LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error)
}

type alertRepository struct{ db *pgxpool.Pool }

func NewAlertRepository(db *pgxpool.Pool) AlertRepository { return &alertRepository{db: db} }

const alertColumns = `id, device_id, client_id, type, severity, message, payload, occurred_at, received_at, created_at`

func scanAlert(row pgx.Row, a *domain.Alert) error {
	return row.Scan(&a.ID, &a.DeviceID, &a.ClientID, &a.Type, &a.Severity, &a.Message, &a.Payload, &a.OccurredAt, &a.ReceivedAt, &a.CreatedAt)
}

func (r *alertRepository) Create(ctx context.Context, a domain.Alert) (domain.Alert, error) {
	query := `INSERT INTO alerts (device_id, client_id, type, severity, message, payload, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING ` + alertColumns
	row := r.db.QueryRow(ctx, query, a.DeviceID, a.ClientID, a.Type, a.Severity, a.Message, a.Payload, a.OccurredAt)
	if err := scanAlert(row, &a); err != nil {
		return domain.Alert{}, mapPgError(err)
	}
	return a, nil
}

func (r *alertRepository) CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error) {
	if len(alerts) == 0 {
		return []domain.Alert{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	results := make([]domain.Alert, 0, len(alerts))
	query := `INSERT INTO alerts (device_id, client_id, type, severity, message, payload, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING ` + alertColumns
	for _, a := range alerts {
		row := tx.QueryRow(ctx, query, a.DeviceID, a.ClientID, a.Type, a.Severity, a.Message, a.Payload, a.OccurredAt)
		var saved domain.Alert
		if err := scanAlert(row, &saved); err != nil {
			return nil, mapPgError(err)
		}
		results = append(results, saved)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *alertRepository) FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error) {
	var a domain.Alert
	row := r.db.QueryRow(ctx, `SELECT `+alertColumns+` FROM alerts WHERE id=$1`, alertID)
	if err := scanAlert(row, &a); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Alert{}, ErrAlertNotFound
		}
		return domain.Alert{}, err
	}
	return a, nil
}

func (r *alertRepository) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	var ts *time.Time
	err := r.db.QueryRow(ctx, `SELECT MAX(occurred_at) FROM alerts WHERE device_id=$1`, deviceID).Scan(&ts)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ErrConstraintViolation
		case "23514":
			return ErrConstraintViolation
		case "22P02":
			if strings.Contains(pgErr.Message, "alert_severity") {
				return ErrConstraintViolation
			}
		}
	}
	return err
}
