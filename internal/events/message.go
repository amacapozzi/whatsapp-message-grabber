package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"msg-grabber/internal/config"
	"msg-grabber/internal/discord"
	"msg-grabber/internal/repository"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	waTypes "go.mau.fi/whatsmeow/types"
	waEvents "go.mau.fi/whatsmeow/types/events"
)

var histStore = repository.NewStore()

type discordSender interface {
	SendMessage(embed discord.Embed) error
	SendMessageWithBytes(embed discord.Embed, filename string, data []byte) error
}

type Handler struct {
	client      *whatsmeow.Client
	discordRepo discordSender
}

const (
	colorPrimary   = 0x5865F2
	colorImage     = 0x00BFFF
	usernameBot    = "WhatsApp Bot"
	historyName    = "history.txt"
	maxCodeBlock   = 1900
	defaultTimeout = 30 * time.Second
)

// 👇 agrega helpers
func (h *Handler) isFromMe(v *waEvents.Message) bool {
	// 1) Si tu versión tiene MessageSource.IsFromMe:
	if v.Info.MessageSource.IsFromMe {
		return true
	}
	// 2) Fallback robusto: comparar el user del sender con tu device user
	return v.Info.Sender.User == h.client.Store.ID.User
}

func phoneFromJID(j waTypes.JID) string {
	// Para contactos 1:1 el server es "s.whatsapp.net"
	if j.Server == waTypes.DefaultUserServer {
		return j.User // número sin '+' (agregalo si querés)
	}
	return ""
}

func (h *Handler) resolveDisplayName(j waTypes.JID) string {
	if h.client == nil || h.client.Store == nil || h.client.Store.Contacts == nil {
		return ""
	}

	ctx := context.Background()
	contact, err := h.client.Store.Contacts.GetContact(ctx, j)
	if err == nil {
		if contact.BusinessName != "" {
			return contact.BusinessName
		}
		if contact.FullName != "" {
			return contact.FullName
		}
		if contact.PushName != "" {
			return contact.PushName
		}
	}

	return ""
}

func buildOutboundEmbed(toName, toNumber string, toJID waTypes.JID, iso, content, myID string) discord.Embed {
	dest := toName
	if dest == "" {
		dest = toNumber
		if dest == "" {
			dest = "(sin nombre)"
		}
	}
	return discord.Embed{
		Username:  usernameBot,
		AvatarURL: "", // si querés el tuyo, ponelo
		Embeds: []discord.EmbedItem{
			{
				Title:       fmt.Sprintf("📤 Mensaje enviado desde %s", myID),
				Description: fmt.Sprintf("**Para:** %s\n**JID:** `%s`", dest, toJID.String()),
				Color:       colorPrimary,
				Timestamp:   iso,
				Fields: []discord.EmbedField{
					{Name: "Contenido", Value: codeBlock(content), Inline: false},
				},
			},
		},
	}
}

func NewMessageEventHandler(client *whatsmeow.Client) func(evt any) {
	h := &Handler{
		client:      client,
		discordRepo: discord.NewDiscordRepository(config.API_CONFIG.WebhookUrl),
	}
	return h.Handle
}

func (h *Handler) Handle(evt any) {
	switch v := evt.(type) {
	case *waEvents.Message:
		h.handleMessage(v)
	case *waEvents.HistorySync:
		h.handleHistorySync(v)
	}
}

func (h *Handler) handleMessage(v *waEvents.Message) {
	push := v.Info.PushName
	jid := v.Info.Sender
	avatar := h.tryGetAvatarURL(jid)

	content := extractTextFromWMI(v.Message)
	if content == "" {
		return
	}

	ts := v.Info.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	iso := ts.UTC().Format(time.RFC3339Nano)

	// ➤ NUEVO: distingue entrante vs saliente
	if h.isFromMe(v) {
		// Saliente: el destinatario es el chat
		toJID := v.Info.Chat
		toName := h.resolveDisplayName(toJID)
		toNumber := phoneFromJID(toJID)

		payload := buildOutboundEmbed(toName, toNumber, toJID, iso, content, h.client.Store.ID.User)
		_ = h.discordRepo.SendMessage(payload)

		// Si querés también guardar en history por destinatario:
		line := fmt.Sprintf("[%s] (yo → %s): %s",
			ts.Format("2006-01-02 15:04:05"),
			func() string {
				if toName != "" {
					return fmt.Sprintf("%s (%s)", toName, toJID.String())
				}
				if toNumber != "" {
					return fmt.Sprintf("+%s (%s)", toNumber, toJID.String())
				}
				return toJID.String()
			}(),
			content,
		)
		_, _ = histStore.AppendBatch(toJID.String(), []string{line})

		// y si hay media saliente, podés replicar lógica de envío si te interesa.
		return
	}

	// Entrante (lo que ya tenías)
	line := fmt.Sprintf("[%s] %s: %s",
		ts.Format("2006-01-02 15:04:05"),
		safe(push),
		content,
	)
	allBytes, _ := histStore.AppendBatch(jid.String(), []string{line})

	payload := buildEmbed(push, avatar, jid, iso, content, h.client.Store.ID.User)

	if len(allBytes) > 0 {
		if err := h.discordRepo.SendMessageWithBytes(payload, historyName, allBytes); err != nil {
			_ = h.discordRepo.SendMessage(payload)
		}
	} else {
		_ = h.discordRepo.SendMessage(payload)
	}

	switch {
	case v.Message.GetAudioMessage() != nil:
		h.sendAudio(v.Message.GetAudioMessage(), ts)
	case v.Message.GetImageMessage() != nil:
		h.sendImage(v.Message.GetImageMessage(), push, avatar, jid, iso)
	case v.Message.DocumentMessage != nil:
		h.sendDocument(v.Message.DocumentMessage)
	case v.Message.VideoMessage != nil:
		h.trySendVideo(v.Message.VideoMessage)
	}
}

func (h *Handler) handleHistorySync(v *waEvents.HistorySync) {
	linesByJID := make(map[string][]string)

	for _, conv := range v.Data.Conversations {
		jid := conv.GetId()
		if jid == "" {
			continue
		}

		for _, hmsg := range conv.Messages {
			if hmsg.Message == nil {
				continue
			}

			wmi := hmsg.Message
			ts := msgTime(wmi)
			sender := msgSender(wmi)
			text := extractTextFromWMI(wmi.Message)
			if text == "" {
				text = "(contenido no textual)"
			}

			line := fmt.Sprintf("[%s] %s: %s",
				ts.Format("2006-01-02 15:04:05"),
				sender,
				text,
			)
			linesByJID[jid] = append(linesByJID[jid], line)
		}
	}

	for jid, lines := range linesByJID {
		if len(lines) == 0 {
			continue
		}
		if _, err := histStore.AppendBatch(jid, lines); err != nil {
			fmt.Println("❌ Error guardando history sync:", err)
		}
	}
}

func (h *Handler) sendAudio(aud *waProto.AudioMessage, ts time.Time) {
	if aud == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	audioBytes, err := h.client.Download(ctx, aud)
	if err != nil {
		fmt.Println("❌ Error descargando audio:", err)
		return
	}

	ext := mimeToExt(aud.GetMimetype())
	if ext == "" {
		ext = "ogg"
	}
	fileName := fmt.Sprintf("audio_%d.%s", ts.Unix(), ext)

	payload := discord.Embed{
		Username: usernameBot,
		Embeds: []discord.EmbedItem{
			{
				Title:       "🎵 Nuevo audio recibido",
				Description: fmt.Sprintf("Archivo subido: `%s` (%d bytes)", fileName, len(audioBytes)),
				Color:       colorPrimary,
			},
		},
	}

	_ = h.discordRepo.SendMessageWithBytes(payload, fileName, audioBytes)
}

func (h *Handler) sendImage(img *waProto.ImageMessage, push, avatar string, jid waTypes.JID, iso string) {
	if img == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	data, err := h.client.Download(ctx, img)
	if err != nil {
		fmt.Println("❌ Error descargando imagen:", err)
		return
	}

	ext := mimeToExt(img.GetMimetype())
	if ext == "" {
		ext = "jpg"
	}
	filename := fmt.Sprintf("image_%d.%s", time.Now().Unix(), ext)

	imgPayload := discord.Embed{
		Username:  safe(push),
		AvatarURL: avatar,
		Embeds: []discord.EmbedItem{
			{
				Title:       fmt.Sprintf("📷 Imagen recibida en %s", h.client.Store.ID.User),
				Description: fmt.Sprintf("**De:** %s\n**JID:** `%s`", safe(push), jid.String()),
				Color:       colorImage,
				Timestamp:   iso,
				Author: &discord.Author{
					Name:    safe(push),
					IconURL: avatar,
				},
				Thumbnail: &discord.Thumbnail{URL: avatar},
			},
		},
	}

	if err := h.discordRepo.SendMessageWithBytes(imgPayload, filename, data); err != nil {
		fmt.Println("❌ Error enviando imagen a Discord:", err)
	}
}

func (h *Handler) sendDocument(doc *waProto.DocumentMessage) {

	payload := discord.Embed{
		Username: usernameBot,
		Embeds: []discord.EmbedItem{
			{
				Title:       "📂 Nuevo archivo recibido",
				Description: fmt.Sprintf("Archivo subido: `%s` (%d bytes)", *doc.FileName, doc.FileLength),
				Color:       colorPrimary,
			},
		},
	}

	docBytes, _ := h.client.Download(context.Background(), doc)

	h.discordRepo.SendMessageWithBytes(payload, *doc.FileName, docBytes)

}

func (h *Handler) trySendVideo(video *waProto.VideoMessage) {
	payload := discord.Embed{
		Username: usernameBot,
		Embeds: []discord.EmbedItem{
			{
				Title:       "📂 Nuevo video recibido",
				Description: fmt.Sprintf("Inoformacion del video: seconds`%d` (%d bytes)", *video.Seconds, *video.FileLength),
				Color:       colorPrimary,
			},
		},
	}

	videoBytes, _ := h.client.Download(context.Background(), video)
	if err := h.discordRepo.SendMessageWithBytes(payload, "video.mp4", videoBytes); err != nil {
		fmt.Println("Error uploading video")
	}
}

func (h *Handler) tryGetAvatarURL(jid waTypes.JID) string {
	if h.client == nil {
		return ""
	}
	if ppi, err := h.client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{Preview: false}); err == nil && ppi != nil {
		return ppi.URL
	}
	return ""
}

func buildEmbed(push, avatar string, jid waTypes.JID, iso, content, id string) discord.Embed {
	return discord.Embed{
		Username:  safe(push),
		AvatarURL: avatar,
		Embeds: []discord.EmbedItem{
			{
				Title:       fmt.Sprintf("📩 Nuevo mensaje recibido en %s", id),
				Description: fmt.Sprintf("**De:** %s\n**JID:** `%s`", safe(push), jid.String()),
				Color:       colorPrimary,
				Timestamp:   iso,
				Author: &discord.Author{
					Name:    safe(push),
					IconURL: avatar,
				},
				Thumbnail: &discord.Thumbnail{URL: avatar},
				Fields: []discord.EmbedField{
					{Name: "Contenido", Value: codeBlock(content), Inline: false},
				},
			},
		},
	}
}

func msgSender(wmi *waProto.WebMessageInfo) string {
	switch {
	case wmi == nil || wmi.Key == nil:
		return "(desconocido)"
	case wmi.GetKey().GetParticipant() != "":
		return wmi.GetKey().GetParticipant()
	case wmi.GetKey().GetFromMe():
		return "(yo)"
	case wmi.GetKey().GetRemoteJid() != "":
		return wmi.GetKey().GetRemoteJid()
	default:
		return "(desconocido)"
	}
}

func msgTime(wmi *waProto.WebMessageInfo) time.Time {
	if wmi == nil {
		return time.Now()
	}
	if sec := wmi.GetMessageTimestamp(); sec > 0 {
		return time.Unix(int64(sec), 0)
	}
	return time.Now()
}

func extractTextFromWMI(msg *waProto.Message) string {
	switch {
	case msg == nil:
		return ""
	case msg.GetConversation() != "":
		return msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil && msg.GetExtendedTextMessage().Text != nil:
		return msg.GetExtendedTextMessage().GetText()
	case msg.GetContactMessage() != nil && msg.GetContactMessage().DisplayName != nil:
		return "[contacto] " + msg.GetContactMessage().GetDisplayName()
	case msg.GetLocationMessage() != nil:
		loc := msg.GetLocationMessage()
		return fmt.Sprintf("[ubicación] lat=%.5f lon=%.5f", loc.GetDegreesLatitude(), loc.GetDegreesLongitude())
	case msg.GetImageMessage() != nil:
		im := msg.GetImageMessage()
		if im.Caption != nil && *im.Caption != "" {
			return "[imagen] " + im.GetCaption()
		}
		return "[imagen]"
	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		return "[video] " + strings.TrimSpace(vm.GetCaption()+" "+vm.GetURL())
	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		name := dm.GetFileName()
		url := dm.GetURL()
		switch {
		case name != "" && url != "":
			return "[documento] " + name + " " + url
		case name != "":
			return "[documento] " + name
		case url != "":
			return "[documento] " + url
		default:
			return "[documento]"
		}
	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		return fmt.Sprintf("[audio] (%d bytes) %s", am.GetFileLength(), strings.TrimSpace(am.GetURL()))
	default:
		return ""
	}
}

func safe(s string) string {
	if s == "" {
		return "(sin nombre)"
	}
	return s
}

func codeBlock(s string) string {
	if len(s) > maxCodeBlock {
		s = s[:maxCodeBlock] + "…"
	}
	return "```\n" + s + "\n```"
}

func mimeToExt(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch mime {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "audio/ogg", "application/ogg":
		return "ogg"
	case "audio/mpeg":
		return "mp3"
	default:
		if i := strings.LastIndex(mime, "/"); i >= 0 {
			return strings.TrimSpace(strings.ToLower(mime[i+1:]))
		}
		return ""
	}
}
