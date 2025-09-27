package github

import (
	"fmt"
	"log"
	"msg-grabber/internal/config"
	"msg-grabber/internal/discord"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func UpdateServerFromWebhook(c *gin.Context) error {

	discordRepo := discord.NewDiscordRepository(config.API_CONFIG.WebhookUrl)

	go func() {
		cmds := []string{
			"git reset --hard && git pull origin main",
			"go build -o server",
			"pm2 restart usermanagement || pm2 start ./server --name usermanagement",
		}

		updateEmbed := discord.Embed{
			Username: "Updated",
			Embeds: []discord.EmbedItem{
				{
					Title:       "New update",
					Description: "Se ha actualizado el codigo",
					Color:       0x00BFFF,
				},
			},
		}

		discordRepo.SendMessage(updateEmbed)

		for _, cmd := range cmds {
			fmt.Println("Executing:", cmd)
			out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
			if err != nil {
				log.Println("❌ Error executing command:", err)
			} else {
				log.Println("✅ OK:", string(out))
			}
		}

	}()

	return nil
}
