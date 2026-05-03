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
// @Summary Create device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body device.CreateDeviceRequest true "Create device request"
// @Success 201 {object} device.CreateDeviceEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 409 {object} common.ErrorResponse
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
// @Summary List devices
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param status query string false "Device status"
// @Param type query string false "Device type"
// @Param include_deleted query bool false "Include deleted devices"
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Success 200 {object} device.ListDevicesResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Router /devices [get]
func (h *deviceHandler) List(c *gin.Context) {
	status := optionalQuery(c, "status")
	deviceType := optionalQuery(c, "type")
	out, err := h.service.ListDevices(c.Request.Context(), currentClientID(c), deviceService.ListDevicesInput{
		Status:         status,
		Type:           deviceType,
		IncludeDeleted: c.Query("include_deleted") == "true",
		Page:           queryInt(c, "page", pagination.DefaultPage),
		PageSize:       queryInt(c, "page_size", pagination.DefaultPageSize),
	})
	if err != nil {
		handleDeviceError(c, err)
		return
	}
	response.Paginated(c, "Devices retrieved successfully", out.Devices, pagination.Meta(out.Page, out.PageSize, out.Total))
}

// Get godoc
// @Summary Get device detail
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} device.DeviceEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
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
// @Summary Update device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Param request body device.UpdateDeviceRequest true "Update device request"
// @Success 200 {object} device.DeviceEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
// @Failure 409 {object} common.ErrorResponse
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
// @Summary Delete device
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} device.DeleteDeviceEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
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
// @Summary Restore deleted device
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} device.DeviceEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
// @Failure 409 {object} common.ErrorResponse
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
// @Summary Rotate device API key
// @Tags Devices
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID"
// @Success 200 {object} device.RotateDeviceAPIKeyEnvelopeResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Failure 404 {object} common.ErrorResponse
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

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func handleDeviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, deviceService.ErrInvalidDeviceType), errors.Is(err, deviceService.ErrInvalidDeviceStatus):
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
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
