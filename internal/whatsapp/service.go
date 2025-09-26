package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func messageEventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())

	}
}

type Service struct {
	Client *whatsmeow.Client
	log    waLog.Logger
}

func New(deviceStore *store.Device, logger waLog.Logger) (*Service, error) {
	if deviceStore == nil {
		return nil, errors.New("deviceStore is nil")
	}
	if logger == nil {
		logger = waLog.Noop
	}
	client := whatsmeow.NewClient(deviceStore, logger)

	s := &Service{Client: client, log: logger}

	client.AddEventHandler(messageEventHandler)

	return s, nil
}

func (s *Service) Connect(_ context.Context) error {
	if s.Client == nil {
		return errors.New("client is nil")
	}
	if s.Client.IsConnected() {
		return nil
	}
	return s.Client.Connect()
}

func (s *Service) GetQRCodePNGBase64(ctx context.Context) (pngBase64 string, alreadyLogged bool, err error) {
	if s.Client == nil {
		return "", false, errors.New("client is nil")
	}

	if s.Client.Store.ID != nil {
		return "", true, nil
	}

	qrChan, err := s.Client.GetQRChannel(ctx)
	if err != nil {
		return "", false, err
	}

	if err := s.Connect(ctx); err != nil {
		return "", false, err
	}

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case evt, ok := <-qrChan:
			if !ok {
				return "", false, errors.New("QR channel closed")
			}
			switch evt.Event {
			case "code":
				const size = 256
				png, encErr := qrcode.Encode(evt.Code, qrcode.Medium, size)
				if encErr != nil {
					return "", false, encErr
				}
				return base64.StdEncoding.EncodeToString(png), false, nil
			case "timeout":
				return "", false, errors.New("QR login timed out")
			case "success":
				return "", false, nil
			}
		case <-timeout.C:
			return "", false, errors.New("timeout waiting for QR code")
		case <-ctx.Done():
			return "", false, ctx.Err()
		}
	}
}

func (s *Service) StartMessageListener() {
	if s.Client == nil {
		return
	}

	s.Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if v.Info.IsFromMe {
				return
			}

			from := v.Info.Sender.String()
			push := v.Info.PushName

			if v.Message.Conversation != nil {
				fmt.Printf("💬 Mensaje de %s (%s): %s\n", push, from, *v.Message.Conversation)
			} else if v.Message.ExtendedTextMessage != nil {
				fmt.Printf("💬 Mensaje extendido de %s: %s\n", push, *v.Message.ExtendedTextMessage.Text)
			} else if v.Message.ImageMessage != nil {
				fmt.Printf("📷 Imagen de %s con caption: %s\n", push, *v.Message.ImageMessage.Caption)
			} else if v.Message.VideoMessage != nil {
				fmt.Printf("🎥 Video de %s con caption: %s\n", push, *v.Message.VideoMessage.Caption)
			} else {
				fmt.Printf("❓ Mensaje no manejado de %s\n", push)
			}
		}
	})
}
