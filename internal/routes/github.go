package routes

import (
	"msg-grabber/internal/github"

	"github.com/gin-gonic/gin"
)

type GithubRoutes struct {
	Engine *gin.Engine
}

func NewGithubRoutes(engine *gin.Engine) *GithubRoutes {
	return &GithubRoutes{Engine: engine}
}

func (r *GithubRoutes) Register() {
	r.Engine.POST("/github/webhook", github.UpdateServerFromWebhook)
}
