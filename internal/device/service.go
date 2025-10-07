package device

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
)

type Service struct {
	deviceRepository DeviceRepository
}

func NewService(d DeviceRepository) *Service {
	return &Service{deviceRepository: d}
}

func (s *Service) GetAllDevices() ([]*store.Device, error) {
	return s.deviceRepository.GetAllDevices(context.Background())
}

func (s *Service) GetDeviceByJid(jidStr string) (*store.Device, error) {
	jid, err := waTypes.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("parse JID: %w", err)
	}
	return s.deviceRepository.GetDeviceByJid(context.Background(), jid)
}
