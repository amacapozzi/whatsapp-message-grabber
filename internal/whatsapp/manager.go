// internal/whatsapp/manager.go
package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"msg-grabber/internal/events"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Manager struct {
	container *sqlstore.Container
	logger    waLog.Logger

	mu       sync.Mutex
	sessions map[string]*Session
}

type Session struct {
	client   *whatsmeow.Client
	dev      *store.Device
	status   string
	deviceID int64
	lastQR   string
	err      error

	ctx    context.Context
	cancel context.CancelFunc
}

func NewManager(container *sqlstore.Container, logger waLog.Logger) *Manager {
	if logger == nil {
		logger = waLog.Noop
	}
	return &Manager{
		container: container,
		logger:    logger,
		sessions:  make(map[string]*Session),
	}
}

type SessionStart struct {
	SessionID string
	FirstQR   string
}

func (m *Manager) CreateSession() (SessionStart, error) {
	if m.container == nil {
		return SessionStart{}, errors.New("container is nil")
	}

	dev := m.container.NewDevice()

	client := whatsmeow.NewClient(dev, m.logger)
	client.EnableAutoReconnect = true

	sctx, cancel := context.WithCancel(context.Background())
	qrChan, err := client.GetQRChannel(sctx)
	if err != nil {
		cancel()
		return SessionStart{}, fmt.Errorf("GetQRChannel: %w", err)
	}

	s := &Session{
		client: client, dev: dev,
		status: "waiting_qr",
		ctx:    sctx, cancel: cancel,
	}
	sid := uuid.NewString()

	m.mu.Lock()
	m.sessions[sid] = s
	m.mu.Unlock()

	go func() {
		if err := client.Connect(); err != nil {
			m.mu.Lock()
			s.status = "error"
			s.err = fmt.Errorf("connect: %w", err)
			m.mu.Unlock()
			return
		}

		for evt := range qrChan {
			switch evt.Event {
			case "code":
				png, encErr := qrcode.Encode(evt.Code, qrcode.Medium, 400)
				m.mu.Lock()
				if encErr != nil {
					s.status = "error"
					s.err = encErr
				} else {
					s.lastQR = base64.StdEncoding.EncodeToString(png)
				}
				m.mu.Unlock()

			case "timeout":
				m.mu.Lock()
				s.status = "timeout"
				m.mu.Unlock()
				return

			case "success":
				j := client.Store.ID
				m.mu.Lock()
				s.deviceID = int64(j.Device)
				s.status = "success"
				s.StartMessageListener()
				m.mu.Unlock()
				return
			}
		}

		m.mu.Lock()
		if s.status == "waiting_qr" {
			s.status = "error"
			s.err = errors.New("QR channel closed")
		}
		m.mu.Unlock()
	}()

	var first string
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		first = m.sessions[sid].lastQR
		m.mu.Unlock()
		if first != "" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return SessionStart{SessionID: sid, FirstQR: first}, nil
}

func (s *Session) StartMessageListener() {
	if s == nil || s.client == nil {
		return
	}
	s.client.AddEventHandler(events.NewMessageEventHandler(s.client))
}

func (m *Manager) GetSessionStatus(sessionID string) (status string, deviceID int64, qrBase64 string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return "", 0, "", errors.New("session not found")
	}
	if s.err != nil && s.status != "success" {
		return s.status, s.deviceID, s.lastQR, s.err
	}
	return s.status, s.deviceID, s.lastQR, nil
}
