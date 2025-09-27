package routes

import (
	"msg-grabber/internal/github"

	"github.com/gin-gonic/gin"
)

type GithubRoutes struct {
	Engine *gin.Engine
}

func (g *GithubRoutes) Register() {
	g.Engine.POST("/github/webhook", func(c *gin.Context) {
		if err := github.UpdateServerFromWebhook(c); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		c.JSON(500, gin.H{"success": "update"})
	})

}
