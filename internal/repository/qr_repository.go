package repository

import (
	"context"

	"go.mau.fi/whatsmeow"
)

type QrRepository interface {
	CreateQrCode(client *whatsmeow.Client, ctx context.Context)
}

func CreateQrCode(client *whatsmeow.Client, ctx context.Context) (string, error) {

	if err := client.Connect(); err != nil {
		return "", nil
	}

	qrChan, _ := client.GetQRChannel(ctx)

	for event := range qrChan {
		if event.Code == "code" {
			return event.Code, nil
		}
	}

	defer client.Disconnect()

	return "", nil
}
