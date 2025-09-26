package main

import (
	initializer "msg-grabber/internal/init"
)

func main() {

	server := initializer.Init()
	server.StartServer()
}
