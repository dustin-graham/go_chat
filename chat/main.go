package main

import (
	"log"
	"os"
)

func main() {
	println("hello")
	logFile, err := os.OpenFile(
		"chat_log.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		// logging is an important part of the app, we've got to bail if we can't do logging
		log.Fatalf("failed to open log file")
	}
	logger := log.New(logFile, "", log.LstdFlags)
	server := NewChatServer(logger)
	if err := server.Start(); err != nil {
		return
	}
	err = server.Start()
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
