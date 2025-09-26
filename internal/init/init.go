package init

import (
	"context"
	"fmt"
	"log"

	"msg-grabber/internal/config"
	"msg-grabber/internal/events"
	"msg-grabber/internal/handlers"
	"msg-grabber/internal/routes"
	"msg-grabber/internal/server"
	"msg-grabber/internal/whatsapp"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var (
	ctx   = context.Background()
	dbLog = waLog.Stdout("Database", "DEBUG", true)
)

func Init() *server.ServerStruct {
	srv := server.NewHTTPServer()

	container, err := sqlstore.New(ctx, "postgres", config.API_CONFIG.DatabaseUrl, dbLog)
	if err != nil {
		log.Panic("sqlstore.New: ", err)
	}

	baseLogger := waLog.Stdout("Client", "INFO", true)

	clients, err := startAllDeviceClients(container, baseLogger)
	if err != nil {
		log.Panic("startAllDeviceClients: ", err)
	}

	var waService *whatsapp.Service
	if len(clients) == 0 {
		deviceStore := container.NewDevice()

		waService, err = whatsapp.New(deviceStore, baseLogger)
		if err != nil {
			log.Panic("whatsapp.New: ", err)
		}

		waService.Client.AddEventHandler(events.NewMessageEventHandler(waService.Client))

		if err := waService.Client.Connect(); err != nil {
			log.Panic("connect (new device): ", err)
		}
	} else {

		firstDev := clients[0]
		waService, err = whatsapp.New(firstDev.Store, baseLogger)
		if err != nil {
			log.Panic("whatsapp.New (from existing device): ", err)
		}
	}

	manager := whatsapp.NewManager(container, baseLogger)

	qrHandler := handlers.NewQrHandler(waService)
	sessionHandler := handlers.NewSessionHandler(manager)

	qrRoutes := routes.NewQrRoutes(srv.Engine, qrHandler, sessionHandler)
	qrRoutes.Register()

	return srv
}

func startAllDeviceClients(container *sqlstore.Container, baseLogger waLog.Logger) ([]*whatsmeow.Client, error) {
	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetAllDevices: %w", err)
	}

	if len(devices) == 0 {
		log.Println("GetAllDevices: no hay devices en DB (se usará flujo QR para agregar uno nuevo).")
		return nil, nil
	}

	var clients []*whatsmeow.Client

	for _, dev := range devices {
		label := "Client"
		if dev.ID != nil {
			label = fmt.Sprintf("Client[%s:%d]", dev.ID.User, dev.ID.Device)
		}
		logger := waLog.Stdout(label, "INFO", true)

		client := whatsmeow.NewClient(dev, logger)
		client.EnableAutoReconnect = true

		client.AddEventHandler(events.NewMessageEventHandler(client))

		if client.Store.ID != nil {
			if err := client.Connect(); err != nil {
				log.Printf("⚠️  Error al conectar %s:%d → %v",
					client.Store.ID.User, client.Store.ID.Device, err)
			} else {
				log.Printf("✅ Conectado en dispositivo: User=%s DeviceID=%d",
					client.Store.ID.User, client.Store.ID.Device)
			}
		} else {
			log.Printf("ℹ️  Device sin JID (DeviceID local=%d). Usar /qr endpoints para loguear.",
				dev.ID.Device)
		}

		clients = append(clients, client)
	}

	return clients, nil
}
