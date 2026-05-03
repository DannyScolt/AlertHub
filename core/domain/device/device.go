package device

import (
	"time"

	"github.com/google/uuid"
)

type DeviceStatus string

const (
	StatusActive      DeviceStatus = "active"
	StatusInactive    DeviceStatus = "inactive"
	StatusMaintenance DeviceStatus = "maintenance"
	StatusError       DeviceStatus = "error"
)

type DeviceType string

const (
	TypeTemperatureSensor DeviceType = "temperature_sensor"
	TypeHumiditySensor    DeviceType = "humidity_sensor"
	TypeSmokeDetector     DeviceType = "smoke_detector"
	TypeMotionSensor      DeviceType = "motion_sensor"
	TypeDoorSensor        DeviceType = "door_sensor"
	TypeCamera            DeviceType = "camera"
	TypeGateway           DeviceType = "gateway"
	TypeOther             DeviceType = "other"
)

type Device struct {
	ID         uuid.UUID
	ClientID   uuid.UUID
	Name       string
	Type       DeviceType
	Status     DeviceStatus
	APIKeyHash string
	Tags       []string
	Metadata   map[string]interface{}
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func (d Device) IsDeleted() bool {
	return d.DeletedAt != nil
}

func ValidStatus(status DeviceStatus) bool {
	switch status {
	case StatusActive, StatusInactive, StatusMaintenance, StatusError:
		return true
	default:
		return false
	}
}

func ValidType(deviceType DeviceType) bool {
	switch deviceType {
	case TypeTemperatureSensor, TypeHumiditySensor, TypeSmokeDetector, TypeMotionSensor, TypeDoorSensor, TypeCamera, TypeGateway, TypeOther:
		return true
	default:
		return false
	}
}

func AllStatuses() []DeviceStatus {
	return []DeviceStatus{StatusActive, StatusInactive, StatusMaintenance, StatusError}
}

func AllTypes() []DeviceType {
	return []DeviceType{TypeTemperatureSensor, TypeHumiditySensor, TypeSmokeDetector, TypeMotionSensor, TypeDoorSensor, TypeCamera, TypeGateway, TypeOther}
}
