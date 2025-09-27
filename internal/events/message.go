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

func NewMessageEventHandler(client *whatsmeow.Client) func(evt interface{}) {
	discordRepo := discord.NewDiscordRepository(config.API_CONFIG.WebhookUrl)

	return func(evt interface{}) {
		switch v := evt.(type) {

		case *waEvents.Message:
			if v.Info.IsFromMe {
				return
			}
			push := v.Info.PushName
			jid := v.Info.Sender

			avatar := tryGetAvatarURL(client, jid)

			content := extractTextFromWMI(v.Message)
			if content == "" {
				return
			}

			ts := v.Info.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			iso := ts.UTC().Format(time.RFC3339Nano)

			line := fmt.Sprintf("[%s] %s: %s",
				ts.Format("2006-01-02 15:04:05"),
				safe(push),
				content,
			)
			allBytes, _ := histStore.AppendBatch(jid.String(), []string{line})

			payload := buildEmbed(push, avatar, jid, iso, content, client.Store.ID.User)

			if len(allBytes) > 0 {
				if err := discordRepo.SendMessageWithBytes(payload, "history.txt", allBytes); err != nil {
					fmt.Println("❌ Error enviando a Discord con adjunto:", err)
					_ = discordRepo.SendMessage(payload)
				}
			} else {
				_ = discordRepo.SendMessage(payload)
			}

			if v.Message.GetImageMessage() != nil {
				img := v.Message.GetImageMessage()

				data, err := client.Download(context.Background(), img)
				if err != nil {
					fmt.Println("❌ Error descargando imagen:", err)
					return
				}

				filename := fmt.Sprintf("image_%d.jpg", ts.Unix())

				imgPayload := discord.Embed{
					Username:  push,
					AvatarURL: avatar,
					Embeds: []discord.EmbedItem{
						{
							Title:       fmt.Sprintf("📷 Imagen recibida en %s", client.Store.ID.User),
							Description: fmt.Sprintf("**De:** %s\n**JID:** `%s`", safe(push), jid.String()),
							Color:       0x00BFFF,
							Timestamp:   iso,
							Author: &discord.Author{
								Name:    safe(push),
								IconURL: avatar,
							},
						},
					},
				}

				if err := discordRepo.SendMessageWithBytes(imgPayload, filename, data); err != nil {
					fmt.Println("❌ Error enviando imagen a Discord:", err)
				}
			}

		case *waEvents.HistorySync:
			linesByJID := map[string][]string{}

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
	}
}

func tryGetAvatarURL(client *whatsmeow.Client, jid waTypes.JID) string {
	if client == nil {
		return ""
	}
	if ppi, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{Preview: false}); err == nil && ppi != nil {
		return ppi.URL
	}
	return ""
}

func buildEmbed(push, avatar string, jid waTypes.JID, iso, content string, id string) discord.Embed {
	return discord.Embed{
		Username:  push,
		AvatarURL: avatar,
		Embeds: []discord.EmbedItem{
			{
				Title:       fmt.Sprintf("📩 Nuevo mensaje recibido en %s", id),
				Description: fmt.Sprintf("**De:** %s\n**JID:** `%s`", safe(push), jid.String()),
				Color:       0x5865F2,
				Timestamp:   iso,
				Author: &discord.Author{
					Name:    safe(push),
					IconURL: avatar,
				},
				Thumbnail: &discord.Thumbnail{
					URL: avatar,
				},
				Fields: []discord.EmbedField{
					{Name: "Contenido", Value: codeBlock(content), Inline: false},
				},
			},
		},
	}
}

func msgSender(wmi *waProto.WebMessageInfo) string {
	if wmi == nil || wmi.Key == nil {
		return "(desconocido)"
	}
	if p := wmi.GetKey().GetParticipant(); p != "" {
		return p
	}
	if wmi.GetKey().GetFromMe() {
		return "(yo)"
	}
	if r := wmi.GetKey().GetRemoteJid(); r != "" {
		return r
	}
	return "(desconocido)"
}

func msgTime(wmi *waProto.WebMessageInfo) time.Time {
	if wmi == nil {
		return time.Now()
	}
	sec := wmi.GetMessageTimestamp()
	if sec <= 0 {
		return time.Now()
	}
	return time.Unix(int64(sec), 0)
}

func extractTextFromWMI(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil && ext.Text != nil {
		return ext.GetText()
	}
	if c := msg.GetContactMessage(); c != nil && c.DisplayName != nil {
		return "[contacto] " + c.GetDisplayName()
	}
	if l := msg.GetLocationMessage(); l != nil {
		return fmt.Sprintf("[ubicación] lat=%.5f lon=%.5f", l.GetDegreesLatitude(), l.GetDegreesLongitude())
	}
	if img := msg.GetImageMessage(); img != nil {
		if img.Caption != nil && *img.Caption != "" {
			return "[imagen] " + img.GetCaption()
		}
		return "[imagen]"
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		return "[video] " + vid.GetCaption()
	}
	if doc := msg.GetDocumentMessage(); doc != nil && doc.FileName != nil {
		return "[documento] " + doc.GetFileName() + doc.GetURL()
	}
	if aud := msg.GetAudioMessage(); aud != nil {
		return fmt.Sprintf("[audio] (%s, %d bytes) AudioURL (%s)", aud.GetMimetype(), aud.GetFileLength(), aud.GetURL())
	}
	return ""
}

func safe(s string) string {
	if s == "" {
		return "(sin nombre)"
	}
	return s
}

func codeBlock(s string) string {
	if len(s) > 1900 {
		s = s[:1900] + "…"
	}
	return "```\n" + s + "\n```"
}

func mimeToExt(mime string) string {
	mime = strings.ToLower(mime)
	switch mime {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		if i := strings.LastIndex(mime, "/"); i >= 0 {
			return strings.Trim(strings.ToLower(mime[i+1:]), " ")
		}
		return ""
	}
}
