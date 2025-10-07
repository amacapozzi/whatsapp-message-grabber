package handlers

import (
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow/store"
)

type ResponseError struct {
	Error string `json:"error"`
}

type DeviceService interface {
	GetAllDevices() ([]*store.Device, error)
	GetDeviceByJid(jid string) (*store.Device, error)
}

type DeviceHandler struct {
	Service DeviceService
}

func NewDeviceHandler(svc DeviceService) *DeviceHandler {
	return &DeviceHandler{Service: svc}
}

func (h *DeviceHandler) GetAllDevices(c *gin.Context) {
	devices, err := h.Service.GetAllDevices()
	if err != nil {
		c.JSON(400, ResponseError{Error: err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": devices})
}

func (h *DeviceHandler) GetDeviceByJid(c *gin.Context) {
	device, err := h.Service.GetDeviceByJid(c.Param("jid"))
	if err != nil {
		c.JSON(400, ResponseError{Error: err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": device})
}
