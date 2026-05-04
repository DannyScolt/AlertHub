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

type ListFilter struct {
	DeviceID   *uuid.UUID
	Severities []domain.Severity
	From       *time.Time
	To         *time.Time
	Search     *string
	Page       int
	PageSize   int
	Offset     int
}

type ListResult struct {
	Alerts []domain.Alert
	Total  int64
}

type IngestRepository interface {
	Create(ctx context.Context, alert domain.Alert) (domain.Alert, error)
	CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error)
}

type QueryRepository interface {
	List(ctx context.Context, clientID uuid.UUID, filter ListFilter) (ListResult, error)
}

type LookupRepository interface {
	FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error)
}

type DeviceActivityRepository interface {
	LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error)
}

type WindowCounter interface {
	ListSameTypeIDsWithinWindow(ctx context.Context, deviceID uuid.UUID, alertType string, since time.Time) ([]uuid.UUID, error)
}

type AlertRepository interface {
	IngestRepository
	QueryRepository
	LookupRepository
	DeviceActivityRepository
	WindowCounter
}

type alertRepository struct{ db *pgxpool.Pool }

func NewAlertRepository(db *pgxpool.Pool) AlertRepository { return &alertRepository{db: db} }

const alertColumns = `id, device_id, client_id, type, severity, message, payload, occurred_at, received_at, created_at`
const qualifiedAlertColumns = `alerts.id, alerts.device_id, alerts.client_id, alerts.type, alerts.severity, alerts.message, alerts.payload, alerts.occurred_at, alerts.received_at, alerts.created_at`

type alertListQueries struct {
	List  string
	Count string
}

func buildAlertListQueries(filter ListFilter) alertListQueries {
	from := "alerts"
	where := `WHERE alerts.client_id = $1
  AND ($2::uuid IS NULL OR alerts.device_id = $2)
  AND ($3::alert_severity[] IS NULL OR alerts.severity = ANY($3::alert_severity[]))
  AND ($4::timestamptz IS NULL OR alerts.occurred_at >= $4)
  AND ($5::timestamptz IS NULL OR alerts.occurred_at <= $5)`
	hasSearchDeviceID := false
	if filter.Search != nil {
		from = "alerts JOIN devices ON devices.id = alerts.device_id AND devices.client_id = $1"
		where += `
  AND (alerts.message ILIKE $6 OR alerts.type ILIKE $6 OR devices.name ILIKE $6`
		if _, err := uuid.Parse(*filter.Search); err == nil {
			hasSearchDeviceID = true
			where += ` OR alerts.device_id = $7`
		}
		where += `)`
	}
	limitPlaceholder := "$6"
	offsetPlaceholder := "$7"
	if filter.Search != nil {
		limitPlaceholder = "$7"
		offsetPlaceholder = "$8"
	}
	if hasSearchDeviceID {
		limitPlaceholder = "$8"
		offsetPlaceholder = "$9"
	}
	return alertListQueries{
		List: `SELECT ` + qualifiedAlertColumns + ` FROM ` + from + `
` + where + `
ORDER BY alerts.occurred_at DESC, alerts.id DESC
LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder,
		Count: `SELECT COUNT(*) FROM ` + from + `
` + where,
	}
}

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

func (r *alertRepository) List(ctx context.Context, clientID uuid.UUID, filter ListFilter) (ListResult, error) {
	severities := severityStrings(filter.Severities)
	queries := buildAlertListQueries(filter)
	args := alertListArgs(clientID, filter, severities)

	rows, err := r.db.Query(ctx, queries.List, append(args, filter.PageSize, filter.Offset)...)
	if err != nil {
		return ListResult{}, mapPgError(err)
	}
	defer rows.Close()

	alerts := make([]domain.Alert, 0)
	for rows.Next() {
		var a domain.Alert
		if err := scanAlert(rows, &a); err != nil {
			return ListResult{}, mapPgError(err)
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, mapPgError(err)
	}

	var total int64
	if err := r.db.QueryRow(ctx, queries.Count, args...).Scan(&total); err != nil {
		return ListResult{}, mapPgError(err)
	}
	return ListResult{Alerts: alerts, Total: total}, nil
}

func alertListArgs(clientID uuid.UUID, filter ListFilter, severities []string) []any {
	args := []any{clientID, filter.DeviceID, severities, filter.From, filter.To}
	if filter.Search == nil {
		return args
	}
	searchPattern := "%" + *filter.Search + "%"
	args = append(args, searchPattern)
	if searchDeviceID, err := uuid.Parse(*filter.Search); err == nil {
		args = append(args, searchDeviceID)
	}
	return args
}

func severityStrings(severities []domain.Severity) []string {
	if len(severities) == 0 {
		return nil
	}
	out := make([]string, 0, len(severities))
	for _, s := range severities {
		out = append(out, string(s))
	}
	return out
}

func (r *alertRepository) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	var ts *time.Time
	err := r.db.QueryRow(ctx, `SELECT MAX(occurred_at) FROM alerts WHERE device_id=$1`, deviceID).Scan(&ts)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func (r *alertRepository) ListSameTypeIDsWithinWindow(ctx context.Context, deviceID uuid.UUID, alertType string, since time.Time) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM alerts WHERE device_id=$1 AND type=$2 AND occurred_at >= $3 ORDER BY occurred_at ASC, id ASC`, deviceID, alertType, since)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapPgError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	return ids, nil
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
