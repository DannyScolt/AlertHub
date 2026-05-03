package alert

import (
	"testing"
	"time"

	domain "alerthub/core/domain/alert"

	"github.com/google/uuid"
)

func TestStreamServiceDispatchesAlertToSameClient(t *testing.T) {
	service := NewStreamService()
	clientID := uuid.New()
	sub := service.Subscribe(clientID, nil)
	defer service.Unsubscribe(sub.ID)
	alert := streamTestAlert(clientID, uuid.New())

	service.Dispatch(alert)

	select {
	case got := <-sub.Alerts:
		if got.ID != alert.ID {
			t.Fatalf("expected alert %s, got %s", alert.ID, got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected alert")
	}
}

func TestStreamServiceDoesNotDispatchToOtherClient(t *testing.T) {
	service := NewStreamService()
	sub := service.Subscribe(uuid.New(), nil)
	defer service.Unsubscribe(sub.ID)

	service.Dispatch(streamTestAlert(uuid.New(), uuid.New()))

	select {
	case got := <-sub.Alerts:
		t.Fatalf("unexpected alert: %+v", got)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestStreamServiceFiltersByDeviceID(t *testing.T) {
	service := NewStreamService()
	clientID := uuid.New()
	wantedDeviceID := uuid.New()
	sub := service.Subscribe(clientID, &wantedDeviceID)
	defer service.Unsubscribe(sub.ID)

	service.Dispatch(streamTestAlert(clientID, uuid.New()))
	service.Dispatch(streamTestAlert(clientID, wantedDeviceID))

	select {
	case got := <-sub.Alerts:
		if got.DeviceID != wantedDeviceID {
			t.Fatalf("expected device %s, got %s", wantedDeviceID, got.DeviceID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected filtered alert")
	}
}

func TestStreamServiceUnsubscribeStopsDelivery(t *testing.T) {
	service := NewStreamService()
	clientID := uuid.New()
	sub := service.Subscribe(clientID, nil)
	service.Unsubscribe(sub.ID)

	service.Dispatch(streamTestAlert(clientID, uuid.New()))

	select {
	case got := <-sub.Alerts:
		t.Fatalf("unexpected alert after unsubscribe: %+v", got)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestStreamServiceDropsOldestWhenSubscriberIsFull(t *testing.T) {
	service := NewStreamServiceWithBuffer(1)
	clientID := uuid.New()
	sub := service.Subscribe(clientID, nil)
	defer service.Unsubscribe(sub.ID)
	first := streamTestAlert(clientID, uuid.New())
	second := streamTestAlert(clientID, uuid.New())

	service.Dispatch(first)
	service.Dispatch(second)

	select {
	case got := <-sub.Alerts:
		if got.ID != second.ID {
			t.Fatalf("expected newest alert %s, got %s", second.ID, got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected newest alert")
	}
}

func streamTestAlert(clientID, deviceID uuid.UUID) domain.Alert {
	now := time.Now().UTC()
	return domain.Alert{ID: uuid.New(), ClientID: clientID, DeviceID: deviceID, Type: "temperature", Severity: domain.SeverityWarning, Message: "Temperature high", Payload: map[string]interface{}{}, OccurredAt: now, ReceivedAt: now, CreatedAt: now}
}
