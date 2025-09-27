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
		github.UpdateServerFromWebhook(c)
	})
}
