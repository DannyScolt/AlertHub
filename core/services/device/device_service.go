package device

import (
	"context"
	"errors"
	"strings"
	"time"

	"alerthub/core/config"
	domain "alerthub/core/domain/device"
	deviceDto "alerthub/core/dto/device"
	alertRepo "alerthub/core/repository/alert"
	deviceRepo "alerthub/core/repository/device"
	"alerthub/core/utils/apikey"
	"alerthub/core/utils/pagination"

	"github.com/google/uuid"
)

var (
	ErrInvalidDeviceType    = errors.New("invalid device type")
	ErrInvalidDeviceStatus  = errors.New("invalid device status")
	ErrDeviceDeleted        = errors.New("device deleted")
	ErrDeviceNotDeleted     = errors.New("device is not deleted")
	ErrRestoreWindowExpired = errors.New("device restore window expired")
)

type ListDevicesInput struct {
	Status         *string
	Type           *string
	IncludeDeleted bool
	Page           int
	PageSize       int
}

type ListDevicesOutput struct {
	Devices  []deviceDto.DeviceResponse
	Total    int64
	Page     int
	PageSize int
}

type DeviceService interface {
	CreateDevice(ctx context.Context, clientID uuid.UUID, req deviceDto.CreateDeviceRequest) (deviceDto.DeviceWithAPIKeyResponse, error)
	ListDevices(ctx context.Context, clientID uuid.UUID, input ListDevicesInput) (ListDevicesOutput, error)
	GetDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error)
	UpdateDevice(ctx context.Context, clientID, deviceID uuid.UUID, req deviceDto.UpdateDeviceRequest) (deviceDto.DeviceResponse, error)
	DeleteDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeleteDeviceResponse, error)
	RestoreDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error)
	RotateAPIKey(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.RotateDeviceAPIKeyResponse, error)
}

type deviceService struct {
	cfg       *config.Config
	repo      deviceRepo.DeviceRepository
	alertRepo alertRepo.AlertRepository
}

func NewDeviceService(cfg *config.Config, repo deviceRepo.DeviceRepository, alerts alertRepo.AlertRepository) DeviceService {
	return &deviceService{cfg: cfg, repo: repo, alertRepo: alerts}
}

func (s *deviceService) CreateDevice(ctx context.Context, clientID uuid.UUID, req deviceDto.CreateDeviceRequest) (deviceDto.DeviceWithAPIKeyResponse, error) {
	dType := domain.DeviceType(req.Type)
	if !domain.ValidType(dType) {
		return deviceDto.DeviceWithAPIKeyResponse{}, ErrInvalidDeviceType
	}
	status := domain.StatusActive
	if strings.TrimSpace(req.Status) != "" {
		status = domain.DeviceStatus(req.Status)
		if !domain.ValidStatus(status) {
			return deviceDto.DeviceWithAPIKeyResponse{}, ErrInvalidDeviceStatus
		}
	}
	exists, err := s.repo.ExistsActiveName(ctx, clientID, req.Name, nil)
	if err != nil {
		return deviceDto.DeviceWithAPIKeyResponse{}, err
	}
	if exists {
		return deviceDto.DeviceWithAPIKeyResponse{}, deviceRepo.ErrDeviceNameConflict
	}
	rawKey, err := apikey.Generate(s.cfg.DeviceAPIKeyPrefix)
	if err != nil {
		return deviceDto.DeviceWithAPIKeyResponse{}, err
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	created, err := s.repo.Create(ctx, domain.Device{ClientID: clientID, Name: req.Name, Type: dType, Status: status, Tags: tags, Metadata: metadata, APIKeyHash: apikey.Hash(rawKey)})
	if err != nil {
		return deviceDto.DeviceWithAPIKeyResponse{}, err
	}
	return deviceDto.DeviceWithAPIKeyResponse{ID: created.ID.String(), Name: created.Name, Type: string(created.Type), Status: string(created.Status), APIKey: rawKey, CreatedAt: created.CreatedAt, UpdatedAt: created.UpdatedAt}, nil
}

func (s *deviceService) ListDevices(ctx context.Context, clientID uuid.UUID, input ListDevicesInput) (ListDevicesOutput, error) {
	page, pageSize := pagination.Normalize(input.Page, input.PageSize)
	filter := deviceRepo.ListFilter{IncludeDeleted: input.IncludeDeleted, Page: page, PageSize: pageSize, Offset: pagination.Offset(page, pageSize)}
	if input.Status != nil && *input.Status != "" {
		status := domain.DeviceStatus(*input.Status)
		if !domain.ValidStatus(status) {
			return ListDevicesOutput{}, ErrInvalidDeviceStatus
		}
		filter.Status = &status
	}
	if input.Type != nil && *input.Type != "" {
		dType := domain.DeviceType(*input.Type)
		if !domain.ValidType(dType) {
			return ListDevicesOutput{}, ErrInvalidDeviceType
		}
		filter.Type = &dType
	}
	result, err := s.repo.List(ctx, clientID, filter)
	if err != nil {
		return ListDevicesOutput{}, err
	}
	devices := make([]deviceDto.DeviceResponse, 0, len(result.Devices))
	for _, d := range result.Devices {
		dto, err := s.toDeviceResponse(ctx, d)
		if err != nil {
			return ListDevicesOutput{}, err
		}
		devices = append(devices, dto)
	}
	return ListDevicesOutput{Devices: devices, Total: result.Total, Page: page, PageSize: pageSize}, nil
}

func (s *deviceService) GetDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error) {
	d, err := s.repo.FindByID(ctx, clientID, deviceID, false)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	return s.toDeviceResponse(ctx, d)
}

func (s *deviceService) UpdateDevice(ctx context.Context, clientID, deviceID uuid.UUID, req deviceDto.UpdateDeviceRequest) (deviceDto.DeviceResponse, error) {
	d, err := s.repo.FindByID(ctx, clientID, deviceID, true)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	if d.IsDeleted() {
		return deviceDto.DeviceResponse{}, ErrDeviceDeleted
	}
	if req.Name != nil {
		exists, err := s.repo.ExistsActiveName(ctx, clientID, *req.Name, &deviceID)
		if err != nil {
			return deviceDto.DeviceResponse{}, err
		}
		if exists {
			return deviceDto.DeviceResponse{}, deviceRepo.ErrDeviceNameConflict
		}
		d.Name = *req.Name
	}
	if req.Type != nil {
		dType := domain.DeviceType(*req.Type)
		if !domain.ValidType(dType) {
			return deviceDto.DeviceResponse{}, ErrInvalidDeviceType
		}
		d.Type = dType
	}
	if req.Status != nil {
		status := domain.DeviceStatus(*req.Status)
		if !domain.ValidStatus(status) {
			return deviceDto.DeviceResponse{}, ErrInvalidDeviceStatus
		}
		d.Status = status
	}
	if req.Tags != nil {
		d.Tags = req.Tags
	}
	if req.Metadata != nil {
		d.Metadata = req.Metadata
	}
	updated, err := s.repo.Update(ctx, d)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	return s.toDeviceResponse(ctx, updated)
}

func (s *deviceService) DeleteDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeleteDeviceResponse, error) {
	d, err := s.repo.SoftDelete(ctx, clientID, deviceID)
	if err != nil {
		return deviceDto.DeleteDeviceResponse{}, err
	}
	return deviceDto.DeleteDeviceResponse{ID: d.ID.String(), DeletedAt: *d.DeletedAt, PurgeAfter: d.DeletedAt.AddDate(0, 0, s.cfg.DeviceRestoreWindowDays)}, nil
}

func (s *deviceService) RestoreDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error) {
	d, err := s.repo.FindByID(ctx, clientID, deviceID, true)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	if !d.IsDeleted() {
		return deviceDto.DeviceResponse{}, ErrDeviceNotDeleted
	}
	if time.Since(*d.DeletedAt) > time.Duration(s.cfg.DeviceRestoreWindowDays)*24*time.Hour {
		return deviceDto.DeviceResponse{}, ErrRestoreWindowExpired
	}
	exists, err := s.repo.ExistsActiveName(ctx, clientID, d.Name, &deviceID)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	if exists {
		return deviceDto.DeviceResponse{}, deviceRepo.ErrDeviceNameConflict
	}
	restored, err := s.repo.Restore(ctx, clientID, deviceID)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	return s.toDeviceResponse(ctx, restored)
}

func (s *deviceService) RotateAPIKey(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.RotateDeviceAPIKeyResponse, error) {
	d, err := s.repo.FindByID(ctx, clientID, deviceID, true)
	if err != nil {
		return deviceDto.RotateDeviceAPIKeyResponse{}, err
	}
	if d.IsDeleted() {
		return deviceDto.RotateDeviceAPIKeyResponse{}, ErrDeviceDeleted
	}
	rawKey, err := apikey.Generate(s.cfg.DeviceAPIKeyPrefix)
	if err != nil {
		return deviceDto.RotateDeviceAPIKeyResponse{}, err
	}
	if err := s.repo.UpdateAPIKeyHash(ctx, clientID, deviceID, apikey.Hash(rawKey)); err != nil {
		return deviceDto.RotateDeviceAPIKeyResponse{}, err
	}
	return deviceDto.RotateDeviceAPIKeyResponse{ID: deviceID.String(), APIKey: rawKey, RotatedAt: time.Now()}, nil
}

func (s *deviceService) toDeviceResponse(ctx context.Context, d domain.Device) (deviceDto.DeviceResponse, error) {
	lastSeen, err := s.alertRepo.LatestOccurredAtByDeviceID(ctx, d.ID)
	if err != nil {
		return deviceDto.DeviceResponse{}, err
	}
	return deviceDto.DeviceResponse{ID: d.ID.String(), Name: d.Name, Type: string(d.Type), Status: string(d.Status), Tags: d.Tags, Metadata: d.Metadata, LastSeenAt: lastSeen, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, DeletedAt: d.DeletedAt}, nil
}
