package device

import (
	"context"
	"fmt"
	"msg-grabber/internal/config"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waTypes "go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var dbLog = waLog.Stdout("Database", "DEBUG", true)

type DeviceDataInteraction struct {
	container *sqlstore.Container
}

type DeviceRepository interface {
	GetAllDevices(ctx context.Context) ([]*store.Device, error)
	GetDeviceByJid(ctx context.Context, jid waTypes.JID) (*store.Device, error)
}

func NewDeviceRepository(cfg *config.Env) (*DeviceDataInteraction, error) {
	if cfg == nil || cfg.DatabaseUrl == "" {
		return nil, fmt.Errorf("missing database url")
	}
	container, err := sqlstore.New(context.Background(), "postgres", cfg.DatabaseUrl, dbLog)
	if err != nil {
		return nil, fmt.Errorf("sqlstore.New: %w", err)
	}
	return &DeviceDataInteraction{container: container}, nil
}

func (d *DeviceDataInteraction) GetAllDevices(ctx context.Context) ([]*store.Device, error) {
	devices, err := d.container.GetAllDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all devices: %w", err)
	}
	return devices, nil
}

func (d *DeviceDataInteraction) GetDeviceByJid(ctx context.Context, jid waTypes.JID) (*store.Device, error) {
	device, err := d.container.GetDevice(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", jid.String(), err)
	}
	return device, nil
}
