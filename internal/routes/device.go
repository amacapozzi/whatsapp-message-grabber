package routes

import (
	"msg-grabber/internal/handlers"

	"github.com/gin-gonic/gin"
)

type DeviceRoutes struct {
	Engine        *gin.Engine
	DeviceHandler *handlers.DeviceHandler
}

func NewDeviceRoutes(engine *gin.Engine, deviceHandler *handlers.DeviceHandler) *DeviceRoutes {
	return &DeviceRoutes{Engine: engine, DeviceHandler: deviceHandler}
}

func (r *DeviceRoutes) DeviceRoutes() {
	r.Engine.GET("/devices", r.DeviceHandler.GetAllDevices)
	r.Engine.GET("/device/:jid", r.DeviceHandler.GetDeviceByJid)
}
