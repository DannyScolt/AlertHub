package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alerthub/core/config"
	domain "alerthub/core/domain/device"
	deviceRepo "alerthub/core/repository/device"
	"alerthub/core/utils/apikey"
	"alerthub/core/utils/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAuthAcceptsRawSwaggerAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	clientID := uuid.New()
	secret := "test-secret"
	accessToken, err := token.GenerateAccessToken(clientID, secret, time.Hour)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	router := gin.New()
	router.Use(Auth(&config.Config{JWTAccessSecret: secret}))
	router.GET("/protected", func(c *gin.Context) {
		got, exists := c.Get(ClientIDKey)
		if !exists {
			t.Fatal("client id missing from context")
		}
		if got != clientID {
			t.Fatalf("client id = %v, want %v", got, clientID)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", accessToken)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNoContent, res.Body.String())
	}
}

func TestDeviceAuthAcceptsRawSwaggerDeviceAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rawKey := "ah_dev_test"
	deviceID := uuid.New()
	clientID := uuid.New()
	repo := &deviceAuthRepositoryStub{device: domain.Device{ID: deviceID, ClientID: clientID}, apiKeyHash: apikey.Hash(rawKey)}

	router := gin.New()
	router.Use(DeviceAuth(&config.Config{}, repo))
	router.POST("/events", func(c *gin.Context) {
		gotDeviceID, exists := c.Get(DeviceIDKey)
		if !exists {
			t.Fatal("device id missing from context")
		}
		if gotDeviceID != deviceID {
			t.Fatalf("device id = %v, want %v", gotDeviceID, deviceID)
		}
		gotClientID, exists := c.Get(ClientIDKey)
		if !exists {
			t.Fatal("client id missing from context")
		}
		if gotClientID != clientID {
			t.Fatalf("client id = %v, want %v", gotClientID, clientID)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/events", nil)
	req.Header.Set("Authorization", rawKey)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusNoContent, res.Body.String())
	}
}

type deviceAuthRepositoryStub struct {
	device     domain.Device
	apiKeyHash string
}

func (s *deviceAuthRepositoryStub) Create(context.Context, domain.Device) (domain.Device, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) ExistsActiveName(context.Context, uuid.UUID, string, *uuid.UUID) (bool, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) FindByID(context.Context, uuid.UUID, uuid.UUID, bool) (domain.Device, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) List(context.Context, uuid.UUID, deviceRepo.ListFilter) (deviceRepo.ListResult, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) Update(context.Context, domain.Device) (domain.Device, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) SoftDelete(context.Context, uuid.UUID, uuid.UUID) (domain.Device, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) Restore(context.Context, uuid.UUID, uuid.UUID) (domain.Device, error) {
	panic("not implemented")
}

func (s *deviceAuthRepositoryStub) FindByAPIKeyHash(_ context.Context, apiKeyHash string) (domain.Device, error) {
	if apiKeyHash != s.apiKeyHash {
		return domain.Device{}, deviceRepo.ErrDeviceNotFound
	}
	return s.device, nil
}

func (s *deviceAuthRepositoryStub) UpdateAPIKeyHash(context.Context, uuid.UUID, uuid.UUID, string) error {
	panic("not implemented")
}
