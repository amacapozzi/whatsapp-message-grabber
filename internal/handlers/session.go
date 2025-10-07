package handlers

import (
	"encoding/base64"
	"net/http"

	"msg-grabber/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	Manager *whatsapp.Manager
}

func NewSessionHandler(m *whatsapp.Manager) *SessionHandler {
	return &SessionHandler{Manager: m}
}

type createSessionResp struct {
	SessionID string `json:"session_id"`
	QRBase64  string `json:"qr_base64,omitempty"`
	Message   string `json:"message,omitempty"`
}

type statusResp struct {
	Status   string `json:"status"`
	DeviceID int64  `json:"device_id,omitempty"`
	QRBase64 string `json:"qr_base64,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	if h.Manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WhatsApp manager not configured"})
		return
	}

	start, err := h.Manager.CreateSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if c.Query("format") == "png" && start.FirstQR != "" {
		b, decErr := base64.StdEncoding.DecodeString(start.FirstQR)
		if decErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": decErr.Error()})
			return
		}
		c.Header("X-Session-ID", start.SessionID)
		c.Data(http.StatusOK, "image/png", b)
		return
	}

	c.JSON(http.StatusOK, createSessionResp{
		SessionID: start.SessionID,
		QRBase64:  start.FirstQR,
		Message:   "Session created; poll /qr/status?session_id=... each 1–2s",
	})
}

func (h *SessionHandler) GetSessionStatus(c *gin.Context) {
	if h.Manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WhatsApp manager not configured"})
		return
	}
	sid := c.Query("session_id")
	if sid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session_id"})
		return
	}

	status, deviceID, qr, err := h.Manager.GetSessionStatus(sid)
	resp := statusResp{Status: status, DeviceID: deviceID, QRBase64: qr}
	if err != nil && status != "success" {
		resp.Error = err.Error()
	}
	c.JSON(http.StatusOK, resp)
}
