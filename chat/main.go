package main

import (
	"log"
)

func main() {
	println("hello")
	server := NewChatServer()
	err := server.Start()
	if err != nil {
		log.Fatalln("error creating server")
	}
	defer func(server *ChatServer) {
		err := server.Stop()
		if err != nil {
			log.Fatalf("error closing server: %v", err)
		}
	}(server)
}
