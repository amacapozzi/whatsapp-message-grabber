package github

import (
	"fmt"
	"log"
	"msg-grabber/internal/config"
	"msg-grabber/internal/discord"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func UpdateServerFromWebhook(c *gin.Context) {
	discordRepo := discord.NewDiscordRepository(config.API_CONFIG.WebhookUrl)

	go func() {
		cmds := []string{
			"git reset --hard && git pull origin main",
			"go build -o server",
			"pm2 restart usermanagement || pm2 start ./server --name usermanagement",
		}

		updateEmbed := discord.Embed{
			Username: "Deploy Bot",
			Embeds: []discord.EmbedItem{
				{
					Title:       "🚀 Nueva actualización",
					Description: "Se ha desplegado el último commit de GitHub.",
					Color:       0x00BFFF,
				},
			},
		}
		_ = discordRepo.SendMessage(updateEmbed)

		//  comandos
		for _, cmd := range cmds {
			fmt.Println("🔹 Ejecutando:", cmd)
			out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
			if err != nil {
				log.Printf("❌ Error en '%s': %v - %s\n", cmd, err, string(out))
			} else {
				log.Printf("✅ OK '%s': %s\n", cmd, string(out))
			}
		}
	}()

	c.JSON(200, gin.H{"success": true, "message": "Update triggered"})
}
