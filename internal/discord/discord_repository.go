package discord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type Embed struct {
	Username  string      `json:"username,omitempty"`
	AvatarURL string      `json:"avatar_url,omitempty"`
	Embeds    []EmbedItem `json:"embeds"`
}

type EmbedItem struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Author      *Author      `json:"author,omitempty"`
	Thumbnail   *Thumbnail   `json:"thumbnail,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
}

type Author struct {
	Name    string `json:"name,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type Thumbnail struct {
	URL string `json:"url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordRepository interface {
	SendMessage(embed Embed) error
	SendMessageWithBytes(embed Embed, filename string, data []byte) error
}

type DiscordDataInteraction struct {
	WebhookURL string
}

func (d *DiscordDataInteraction) SendMessage(embed Embed) error {
	if d.WebhookURL == "" {
		return errors.New("webhook url is empty")
	}

	jsonData, _ := json.Marshal(embed)
	resp, err := http.Post(d.WebhookURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error: %s: %s", resp.Status, string(b))
	}
	return nil
}

func (d *DiscordDataInteraction) SendMessageWithBytes(embed Embed, filename string, data []byte) error {
	if d.WebhookURL == "" {
		return errors.New("webhook url is empty")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	payload, err := json.Marshal(embed)
	if err != nil {
		return fmt.Errorf("marshal embed: %w", err)
	}
	if err := writer.WriteField("payload_json", string(payload)); err != nil {
		return fmt.Errorf("write payload_json: %w", err)
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", d.WebhookURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error: %s: %s", resp.Status, string(b))
	}
	return nil
}

func NewDiscordRepository(webhookUrl string) DiscordRepository {
	return &DiscordDataInteraction{WebhookURL: webhookUrl}
}
