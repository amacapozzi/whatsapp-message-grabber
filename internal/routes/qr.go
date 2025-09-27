package routes

import (
	"msg-grabber/internal/handlers"

	"github.com/gin-gonic/gin"
)

type QrRoutes struct {
	Engine         *gin.Engine
	QrHandler      *handlers.QrHandler
	SessionHandler *handlers.SessionHandler
}

func NewQrRoutes(engine *gin.Engine, qh *handlers.QrHandler, sh *handlers.SessionHandler) *QrRoutes {
	return &QrRoutes{
		Engine:         engine,
		QrHandler:      qh,
		SessionHandler: sh,
	}
}

func (r *QrRoutes) Register() {
	r.Engine.GET("/qr", r.QrHandler.GetQrCode)
	r.Engine.GET("/qr/create-session", r.SessionHandler.CreateSession)
}
