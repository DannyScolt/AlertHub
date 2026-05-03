package device

import (
	"context"
	"testing"
	"time"

	"alerthub/core/config"
	alertDomain "alerthub/core/domain/alert"
	domain "alerthub/core/domain/device"
	deviceDto "alerthub/core/dto/device"
	deviceRepo "alerthub/core/repository/device"

	"github.com/google/uuid"
)

type createDeviceRepoStub struct {
	created domain.Device
}

func (r *createDeviceRepoStub) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	r.created = device
	device.ID = uuid.New()
	return device, nil
}

func (r *createDeviceRepoStub) ExistsActiveName(ctx context.Context, clientID uuid.UUID, name string, excludeDeviceID *uuid.UUID) (bool, error) {
	return false, nil
}

func (r *createDeviceRepoStub) FindByID(ctx context.Context, clientID, deviceID uuid.UUID, includeDeleted bool) (domain.Device, error) {
	return domain.Device{}, deviceRepo.ErrDeviceNotFound
}

func (r *createDeviceRepoStub) FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (domain.Device, error) {
	return domain.Device{}, deviceRepo.ErrDeviceNotFound
}

func (r *createDeviceRepoStub) List(ctx context.Context, clientID uuid.UUID, filter deviceRepo.ListFilter) (deviceRepo.ListResult, error) {
	return deviceRepo.ListResult{}, nil
}

func (r *createDeviceRepoStub) Update(ctx context.Context, device domain.Device) (domain.Device, error) {
	return domain.Device{}, nil
}

func (r *createDeviceRepoStub) SoftDelete(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error) {
	return domain.Device{}, nil
}

func (r *createDeviceRepoStub) Restore(ctx context.Context, clientID, deviceID uuid.UUID) (domain.Device, error) {
	return domain.Device{}, nil
}

func (r *createDeviceRepoStub) UpdateAPIKeyHash(ctx context.Context, clientID, deviceID uuid.UUID, apiKeyHash string) error {
	return nil
}

type deviceAlertRepoStub struct{}

func (r deviceAlertRepoStub) Create(ctx context.Context, alert alertDomain.Alert) (alertDomain.Alert, error) {
	return alertDomain.Alert{}, nil
}

func (r deviceAlertRepoStub) CreateBatch(ctx context.Context, alerts []alertDomain.Alert) ([]alertDomain.Alert, error) {
	return nil, nil
}

func (r deviceAlertRepoStub) FindByID(ctx context.Context, alertID uuid.UUID) (alertDomain.Alert, error) {
	return alertDomain.Alert{}, nil
}

func (r deviceAlertRepoStub) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

func TestCreateDeviceDefaultsOptionalTagsAndMetadata(t *testing.T) {
	repo := &createDeviceRepoStub{}
	service := NewDeviceService(&config.Config{DeviceAPIKeyPrefix: "ah_test"}, repo, deviceAlertRepoStub{})

	_, err := service.CreateDevice(context.Background(), uuid.New(), deviceDto.CreateDeviceRequest{
		Name: "Minimal Device",
		Type: string(domain.TypeTemperatureSensor),
	})
	if err != nil {
		t.Fatalf("CreateDevice returned error: %v", err)
	}
	if repo.created.Tags == nil {
		t.Fatal("expected empty tags slice, got nil")
	}
	if repo.created.Metadata == nil {
		t.Fatal("expected empty metadata map, got nil")
	}
}
