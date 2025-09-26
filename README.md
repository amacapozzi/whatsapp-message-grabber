# WhatsApp Message Grabber

Servicio en Go que:

- Inicia sesión en **WhatsApp Web** usando [whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow).
- **Escucha mensajes** entrantes (y procesa `HistorySync` para mensajes viejos).
- Envía notificaciones a **Discord** vía webhook:
  - Embed con datos del mensaje (remitente, hora, avatar).
  - Adjunta **history.txt** con TODO el historial del chat (acumulado).
  - Si el mensaje trae **imagen**, también la adjunta al webhook.

## 📦 Stack

- Go 1.21+
- Postgres (almacenamiento de dispositivos de whatsmeow)
- Gin (API)
- Whatsmeow (cliente WA)
- Discord Webhook

## 🗂 Estructura (simplificada)
