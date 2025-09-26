package server

import (
	"log"

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
	return &ServerStruct{Engine: e}
}
