package server

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type ServerStruct struct {
	Engine *gin.Engine
}

func (s *ServerStruct) StartServer() {
	if err := s.Engine.Run(); err != nil {
		log.Panic(err)
	}
}

func NewHTTPServer() *ServerStruct {
	e := gin.Default()

	e.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return &ServerStruct{Engine: e}
}
