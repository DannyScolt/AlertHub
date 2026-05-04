package device

import (
	"errors"
	"net/http"
	"strconv"

	deviceDto "alerthub/core/dto/device"
	"alerthub/core/middleware"
	deviceRepo "alerthub/core/repository/device"
	deviceService "alerthub/core/services/device"
	"alerthub/core/utils/pagination"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DeviceHandler interface {
	Create(*gin.Context)
	List(*gin.Context)
	Get(*gin.Context)
	Update(*gin.Context)
	Delete(*gin.Context)
	Restore(*gin.Context)
	RotateAPIKey(*gin.Context)
}

type deviceHandler struct{ service deviceService.DeviceService }

func NewDeviceHandler(service deviceService.DeviceService) DeviceHandler {
	return &deviceHandler{service: service}
}

// Create godoc
// @Summary Register a new device for the authenticated client
// @Description Creates an IoT device owned by the current client. `type` must be one of: temperature_sensor, humidity_sensor, smoke_detector, motion_sensor, door_sensor, camera, gateway, other. `status` is optional and defaults to active; allowed values are active, inactive, maintenance, error. Device names must be unique per client among non-deleted devices. The generated api_key is returned only once in this response and only its hash is stored.
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body device.CreateDeviceRequest true "Device registration payload. tags and metadata are optional."
// @Success 201 {object} device.CreateDeviceEnvelopeResponse "Device created successfully; save data.api_key because it cannot be retrieved later."
// @Failure 400 {object} common.ErrorResponse "Validation error, invalid device type, or invalid status."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 409 {object} common.ErrorResponse "A non-deleted device with the same name already exists for this client."
// @Router /devices [post]
func (h *deviceHandler) Create(c *gin.Context) {
	var req deviceDto.CreateDeviceRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.CreateDevice(c.Request.Context(), currentClientID(c), req)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Device created successfully", data)
}

// List godoc
// @Summary List devices owned by the authenticated client
// @Description Returns paginated devices for the current client. By default soft-deleted devices are hidden. Use `status` to satisfy Backlog 1 filtering by device status; combine with `type` when needed. Allowed status values: active, inactive, maintenance, error. Allowed type values: temperature_sensor, humidity_sensor, smoke_detector, motion_sensor, door_sensor, camera, gateway, other.
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by device status: active, inactive, maintenance, error" Enums(active,inactive,maintenance,error)
// @Param type query string false "Filter by device type" Enums(temperature_sensor,humidity_sensor,smoke_detector,motion_sensor,door_sensor,camera,gateway,other)
// @Param include_deleted query bool false "Set true to include soft-deleted devices" default(false)
// @Param page query int false "Page number, starts at 1" minimum(1) default(1)
// @Param page_size query int false "Items per page" minimum(1) maximum(100) default(20)
// @Success 200 {object} device.ListDevicesResponse "Devices retrieved successfully with pagination metadata."
// @Failure 400 {object} common.ErrorResponse "Invalid status, type, or pagination input."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Router /devices [get]
func (h *deviceHandler) List(c *gin.Context) {
	input, err := listDevicesInput(c)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	out, err := h.service.ListDevices(c.Request.Context(), currentClientID(c), input)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Paginated(c, "Devices retrieved successfully", out.Devices, pagination.Meta(out.Page, out.PageSize, out.Total))
}

// Get godoc
// @Summary Get one device by ID
// @Description Returns a non-deleted device owned by the authenticated client. Soft-deleted devices are not returned by this endpoint unless they are restored first.
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Success 200 {object} device.DeviceEnvelopeResponse "Device retrieved successfully."
// @Failure 400 {object} common.ErrorResponse "Invalid device id."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Device not found for this client or device is soft-deleted."
// @Router /devices/{id} [get]
func (h *deviceHandler) Get(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	data, err := h.service.GetDevice(c.Request.Context(), currentClientID(c), deviceID)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device retrieved successfully", data)
}

// Update godoc
// @Summary Update a device
// @Description Partially updates a non-deleted device owned by the authenticated client. Send only the fields you want to change. Allowed status values: active, inactive, maintenance, error. Allowed type values: temperature_sensor, humidity_sensor, smoke_detector, motion_sensor, door_sensor, camera, gateway, other. Device name uniqueness is enforced per client among non-deleted devices.
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Param request body device.UpdateDeviceRequest true "Partial update payload. Omitted fields keep their existing values."
// @Success 200 {object} device.DeviceEnvelopeResponse "Device updated successfully."
// @Failure 400 {object} common.ErrorResponse "Invalid device id, invalid type/status, validation error, or deleted device cannot be changed."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Device not found for this client."
// @Failure 409 {object} common.ErrorResponse "Updated name conflicts with another active device owned by the client."
// @Router /devices/{id} [patch]
func (h *deviceHandler) Update(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	var req deviceDto.UpdateDeviceRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.UpdateDevice(c.Request.Context(), currentClientID(c), deviceID, req)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device updated successfully", data)
}

// Delete godoc
// @Summary Soft delete a device
// @Description Soft deletes a device owned by the authenticated client by setting deleted_at. Deleted devices are hidden from normal list/detail APIs. The response includes purge_after, which is the point after the configured restore window.
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Success 200 {object} device.DeleteDeviceEnvelopeResponse "Device deleted successfully; data.deleted_at and data.purge_after describe the deletion lifecycle."
// @Failure 400 {object} common.ErrorResponse "Invalid device id."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Device not found for this client."
// @Router /devices/{id} [delete]
func (h *deviceHandler) Delete(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	data, err := h.service.DeleteDevice(c.Request.Context(), currentClientID(c), deviceID)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device deleted successfully", data)
}

// Restore godoc
// @Summary Restore a soft-deleted device
// @Description Restores a device deleted within the configured restore window. Restored devices are set to inactive so the client can review them before using them again. Restore can fail if another active device now uses the same name.
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Success 200 {object} device.DeviceEnvelopeResponse "Device restored successfully; restored status is inactive."
// @Failure 400 {object} common.ErrorResponse "Invalid device id, device is not deleted, or restore window expired."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Device not found for this client."
// @Failure 409 {object} common.ErrorResponse "Restore conflicts with another active device name."
// @Router /devices/{id}/restore [post]
func (h *deviceHandler) Restore(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	data, err := h.service.RestoreDevice(c.Request.Context(), currentClientID(c), deviceID)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device restored successfully", data)
}

// RotateAPIKey godoc
// @Summary Rotate a device API key
// @Description Generates a new API key for a non-deleted device owned by the authenticated client and replaces the stored hash. The raw api_key is returned only once in this response; store it immediately because it cannot be retrieved later.
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Success 200 {object} device.RotateDeviceAPIKeyEnvelopeResponse "Device API key rotated successfully."
// @Failure 400 {object} common.ErrorResponse "Invalid device id or deleted device cannot be changed."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Device not found for this client."
// @Router /devices/{id}/rotate-key [post]
func (h *deviceHandler) RotateAPIKey(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	data, err := h.service.RotateAPIKey(c.Request.Context(), currentClientID(c), deviceID)
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device API key rotated successfully", data)
}

func currentClientID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.ClientIDKey).(uuid.UUID)
}

func parseDeviceID(c *gin.Context) (uuid.UUID, bool) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid device id", nil)
		return uuid.Nil, false
	}
	return deviceID, true
}

func bind(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", err.Error())
		return false
	}
	return true
}

func optionalQuery(c *gin.Context, key string) *string {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

func listDevicesInput(c *gin.Context) (deviceService.ListDevicesInput, error) {
	page, err := queryInt(c, "page", pagination.DefaultPage)
	if err != nil {
		return deviceService.ListDevicesInput{}, err
	}
	pageSize, err := queryInt(c, "page_size", pagination.DefaultPageSize)
	if err != nil {
		return deviceService.ListDevicesInput{}, err
	}
	return deviceService.ListDevicesInput{
		Status:         optionalQuery(c, "status"),
		Type:           optionalQuery(c, "type"),
		IncludeDeleted: c.Query("include_deleted") == "true",
		Page:           page,
		PageSize:       pageSize,
	}, nil
}

func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, deviceService.ErrInvalidPagination
	}
	return value, nil
}

func handleDeviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, deviceService.ErrInvalidDeviceType), errors.Is(err, deviceService.ErrInvalidDeviceStatus):
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, deviceService.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, "INVALID_PAGINATION", "Invalid pagination parameters", nil)
	case errors.Is(err, deviceRepo.ErrDeviceNameConflict):
		response.Error(c, http.StatusConflict, "DEVICE_NAME_CONFLICT", "Device name already exists", nil)
	case errors.Is(err, deviceRepo.ErrDeviceNotFound):
		response.Error(c, http.StatusNotFound, "DEVICE_NOT_FOUND", "Device not found", nil)
	case errors.Is(err, deviceService.ErrDeviceDeleted):
		response.Error(c, http.StatusBadRequest, "DEVICE_DELETED", "Deleted devices cannot be changed", nil)
	case errors.Is(err, deviceService.ErrDeviceNotDeleted):
		response.Error(c, http.StatusBadRequest, "DEVICE_NOT_DELETED", "Device is not deleted", nil)
	case errors.Is(err, deviceService.ErrRestoreWindowExpired):
		response.Error(c, http.StatusBadRequest, "RESTORE_WINDOW_EXPIRED", "Device restore window expired", nil)
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
