package alert

import (
	"sync"

	domain "alerthub/core/domain/alert"

	"github.com/google/uuid"
)

const defaultSubscriberBuffer = 32

type StreamService interface {
	Subscribe(clientID uuid.UUID, deviceID *uuid.UUID) Subscriber
	Unsubscribe(subscriberID uuid.UUID)
	Dispatch(alert domain.Alert)
}

type Subscriber struct {
	ID       uuid.UUID
	ClientID uuid.UUID
	DeviceID *uuid.UUID
	Alerts   chan domain.Alert
}

type streamService struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]Subscriber
	bufferSize  int
}

func NewStreamService() StreamService {
	return NewStreamServiceWithBuffer(defaultSubscriberBuffer)
}

func NewStreamServiceWithBuffer(bufferSize int) StreamService {
	return &streamService{subscribers: map[uuid.UUID]Subscriber{}, bufferSize: bufferSize}
}

func (s *streamService) Subscribe(clientID uuid.UUID, deviceID *uuid.UUID) Subscriber {
	sub := Subscriber{ID: uuid.New(), ClientID: clientID, DeviceID: deviceID, Alerts: make(chan domain.Alert, s.bufferSize)}
	s.mu.Lock()
	s.subscribers[sub.ID] = sub
	s.mu.Unlock()
	return sub
}

func (s *streamService) Unsubscribe(subscriberID uuid.UUID) {
	s.mu.Lock()
	delete(s.subscribers, subscriberID)
	s.mu.Unlock()
}

func (s *streamService) Dispatch(alert domain.Alert) {
	s.mu.RLock()
	subscribers := make([]Subscriber, 0, len(s.subscribers))
	for _, sub := range s.subscribers {
		subscribers = append(subscribers, sub)
	}
	s.mu.RUnlock()

	for _, sub := range subscribers {
		if sub.ClientID != alert.ClientID {
			continue
		}
		if sub.DeviceID != nil && *sub.DeviceID != alert.DeviceID {
			continue
		}
		select {
		case sub.Alerts <- alert:
		default:
			select {
			case <-sub.Alerts:
			default:
			}
			sub.Alerts <- alert
		}
	}
}
