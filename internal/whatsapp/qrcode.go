package whatsapp

import (
	"encoding/base64"

	"github.com/skip2/go-qrcode"
)

func EncodeQRToPNGBase64(data string) (string, error) {
	const pngSize = 256
	pngBytes, err := qrcode.Encode(data, qrcode.Medium, pngSize)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngBytes), nil
}
