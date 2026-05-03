package device

import (
	"context"
	"errors"
	"strconv"
	"strings"

	domain "alerthub/core/domain/device"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceNameConflict  = errors.New("device name already exists")
	ErrConstraintViolation = errors.New("constraint violation")
)

type ListFilter struct {
	Status         *domain.DeviceStatus
	Type           *domain.DeviceType
	IncludeDeleted bool
	Page           int
	PageSize       int
	Offset         int
}

type ListResult struct {
	Devices []domain.Device
	Total   int64
}

type DeviceRepository interface {
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	FindByID(ctx context.Context, clientID, deviceID uuid.UUID, includeDeleted bool) (domain.Device, error)
	FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (domain.Device, error)
	List(ctx context.Context, clientID uuid.UUID, filter ListFilter) (ListResult, error)
	Update(ctx context.Context, device domain.Device) (domain.Device, error)
	SoftDelete(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error)
	Restore(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error)
	UpdateAPIKeyHash(ctx context.Context, clientID, deviceID uuid.UUID, apiKeyHash string) error
	ExistsActiveName(ctx context.Context, clientID uuid.UUID, name string, excludeDeviceID *uuid.UUID) (bool, error)
}

type deviceRepository struct{ db *pgxpool.Pool }

func NewDeviceRepository(db *pgxpool.Pool) DeviceRepository { return &deviceRepository{db: db} }

const deviceColumns = `id, client_id, name, type, status, api_key_hash, tags, metadata, created_at, updated_at, deleted_at`

func scanDevice(row pgx.Row, d *domain.Device) error {
	return row.Scan(&d.ID, &d.ClientID, &d.Name, &d.Type, &d.Status, &d.APIKeyHash, &d.Tags, &d.Metadata, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt)
}

func (r *deviceRepository) Create(ctx context.Context, d domain.Device) (domain.Device, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO devices (client_id, name, type, status, api_key_hash, tags, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING `+deviceColumns, d.ClientID, d.Name, d.Type, d.Status, d.APIKeyHash, d.Tags, d.Metadata)
	if err := scanDevice(row, &d); err != nil {
		return domain.Device{}, mapPgError(err)
	}
	return d, nil
}

func (r *deviceRepository) FindByID(ctx context.Context, clientID, deviceID uuid.UUID, includeDeleted bool) (domain.Device, error) {
	query := `SELECT ` + deviceColumns + ` FROM devices WHERE client_id=$1 AND id=$2`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	var d domain.Device
	if err := scanDevice(r.db.QueryRow(ctx, query, clientID, deviceID), &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, ErrDeviceNotFound
		}
		return domain.Device{}, err
	}
	return d, nil
}

func (r *deviceRepository) FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (domain.Device, error) {
	var d domain.Device
	if err := scanDevice(r.db.QueryRow(ctx, `SELECT `+deviceColumns+` FROM devices WHERE api_key_hash=$1 AND deleted_at IS NULL`, apiKeyHash), &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, ErrDeviceNotFound
		}
		return domain.Device{}, err
	}
	return d, nil
}

func (r *deviceRepository) List(ctx context.Context, clientID uuid.UUID, filter ListFilter) (ListResult, error) {
	conditions := []string{`client_id=$1`}
	args := []interface{}{clientID}
	argPos := 2
	if !filter.IncludeDeleted {
		conditions = append(conditions, `deleted_at IS NULL`)
	}
	if filter.Status != nil {
		conditions = append(conditions, `status=$`+strconv.Itoa(argPos))
		args = append(args, *filter.Status)
		argPos++
	}
	if filter.Type != nil {
		conditions = append(conditions, `type=$`+strconv.Itoa(argPos))
		args = append(args, *filter.Type)
		argPos++
	}
	where := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM devices WHERE `+where, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}

	args = append(args, filter.PageSize, filter.Offset)
	query := `SELECT ` + deviceColumns + ` FROM devices WHERE ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	devices := make([]domain.Device, 0)
	for rows.Next() {
		var d domain.Device
		if err := scanDevice(rows, &d); err != nil {
			return ListResult{}, err
		}
		devices = append(devices, d)
	}
	return ListResult{Devices: devices, Total: total}, rows.Err()
}

func (r *deviceRepository) Update(ctx context.Context, d domain.Device) (domain.Device, error) {
	row := r.db.QueryRow(ctx, `UPDATE devices SET name=$3, type=$4, status=$5, tags=$6, metadata=$7, updated_at=NOW() WHERE client_id=$1 AND id=$2 RETURNING `+deviceColumns, d.ClientID, d.ID, d.Name, d.Type, d.Status, d.Tags, d.Metadata)
	if err := scanDevice(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, ErrDeviceNotFound
		}
		return domain.Device{}, mapPgError(err)
	}
	return d, nil
}

func (r *deviceRepository) SoftDelete(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error) {
	var d domain.Device
	row := r.db.QueryRow(ctx, `UPDATE devices SET deleted_at=COALESCE(deleted_at, NOW()), updated_at=NOW() WHERE client_id=$1 AND id=$2 RETURNING `+deviceColumns, clientID, deviceID)
	if err := scanDevice(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, ErrDeviceNotFound
		}
		return domain.Device{}, err
	}
	return d, nil
}

func (r *deviceRepository) Restore(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error) {
	var d domain.Device
	row := r.db.QueryRow(ctx, `UPDATE devices SET deleted_at=NULL, status='inactive', updated_at=NOW() WHERE client_id=$1 AND id=$2 RETURNING `+deviceColumns, clientID, deviceID)
	if err := scanDevice(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, ErrDeviceNotFound
		}
		return domain.Device{}, mapPgError(err)
	}
	return d, nil
}

func (r *deviceRepository) UpdateAPIKeyHash(ctx context.Context, clientID, deviceID uuid.UUID, apiKeyHash string) error {
	cmd, err := r.db.Exec(ctx, `UPDATE devices SET api_key_hash=$3, updated_at=NOW() WHERE client_id=$1 AND id=$2 AND deleted_at IS NULL`, clientID, deviceID, apiKeyHash)
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *deviceRepository) ExistsActiveName(ctx context.Context, clientID uuid.UUID, name string, excludeDeviceID *uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE client_id=$1 AND lower(name)=lower($2) AND deleted_at IS NULL`
	args := []interface{}{clientID, name}
	if excludeDeviceID != nil {
		query += ` AND id<>$3`
		args = append(args, *excludeDeviceID)
	}
	query += `)`
	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && pgErr.ConstraintName == "idx_devices_client_name_active_unique" {
			return ErrDeviceNameConflict
		}
		if pgErr.Code == "23505" {
			return ErrConstraintViolation
		}
	}
	return err
}
