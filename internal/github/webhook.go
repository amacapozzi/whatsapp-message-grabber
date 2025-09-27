package github

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func UpdateServerFromWebhook(c *gin.Context) error {
	go func() {
		cmds := []string{
			"git reset --hard && git pull origin main",
			"pm2 restart all",
		}
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
