package handlers

import (
	"context"
	"msg-grabber/internal/whatsapp"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type QrHandler struct {
	WA *whatsapp.Service
}

func NewQrHandler(wa *whatsapp.Service) *QrHandler {
	return &QrHandler{WA: wa}
}

type qrResponse struct {
	AlreadyLogged bool   `json:"alreadyLogged"`
	PNGBase64     string `json:"pngBase64,omitempty"`
	Message       string `json:"message,omitempty"`
}

func (h *QrHandler) GetQrCode(c *gin.Context) {
	if h.WA == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WhatsApp service not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 35*time.Second)
	defer cancel()

	png, already, err := h.WA.GetQRCodePNGBase64(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, qrResponse{
		AlreadyLogged: already,
		PNGBase64:     png,
		Message:       "OK",
	})
}
