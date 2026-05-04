package device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	deviceDto "alerthub/core/dto/device"
	"alerthub/core/middleware"
	deviceService "alerthub/core/services/device"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type deviceServiceStub struct{}

func (s *deviceServiceStub) CreateDevice(ctx context.Context, clientID uuid.UUID, req deviceDto.CreateDeviceRequest) (deviceDto.DeviceWithAPIKeyResponse, error) {
	return deviceDto.DeviceWithAPIKeyResponse{}, nil
}

func (s *deviceServiceStub) ListDevices(ctx context.Context, clientID uuid.UUID, input deviceService.ListDevicesInput) (deviceService.ListDevicesOutput, error) {
	return deviceService.ListDevicesOutput{Devices: []deviceDto.DeviceResponse{}, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *deviceServiceStub) GetDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error) {
	return deviceDto.DeviceResponse{}, nil
}

func (s *deviceServiceStub) UpdateDevice(ctx context.Context, clientID, deviceID uuid.UUID, req deviceDto.UpdateDeviceRequest) (deviceDto.DeviceResponse, error) {
	return deviceDto.DeviceResponse{}, nil
}

func (s *deviceServiceStub) DeleteDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeleteDeviceResponse, error) {
	return deviceDto.DeleteDeviceResponse{}, nil
}

func (s *deviceServiceStub) RestoreDevice(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.DeviceResponse, error) {
	return deviceDto.DeviceResponse{}, nil
}

func (s *deviceServiceStub) RotateAPIKey(ctx context.Context, clientID, deviceID uuid.UUID) (deviceDto.RotateDeviceAPIKeyResponse, error) {
	return deviceDto.RotateDeviceAPIKeyResponse{}, nil
}

func performListDevicesRequest(path string, service deviceService.DeviceService) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDeviceHandler(service)
	router.GET("/devices", func(c *gin.Context) {
		c.Set(middleware.ClientIDKey, uuid.New())
		handler.List(c)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestListRejectsNonNumericPage(t *testing.T) {
	recorder := performListDevicesRequest("/devices?page=abc", &deviceServiceStub{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestListRejectsNonNumericPageSize(t *testing.T) {
	recorder := performListDevicesRequest("/devices?page_size=abc", &deviceServiceStub{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}
