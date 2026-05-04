package device

import (
	"context"
	"testing"
	"time"

	"alerthub/core/config"
	domain "alerthub/core/domain/device"
	deviceDto "alerthub/core/dto/device"
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

type deviceAlertRepoStub struct{}

func (r deviceAlertRepoStub) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

func TestCreateDeviceDefaultsOptionalTagsAndMetadata(t *testing.T) {
	repo := &createDeviceRepoStub{}
	service := NewFocusedDeviceService(&config.Config{DeviceAPIKeyPrefix: "ah_test"}, repo, nil, nil, nil, nil, deviceAlertRepoStub{})

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
