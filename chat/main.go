package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logFile, err := os.OpenFile(
		"chat_log.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		// logging is an important part of the app, we've got to bail if we can't do logging
		log.Fatalf("failed to open log file")
	}

	// chat chatServer
	logger := log.New(logFile, "", log.LstdFlags)
	chatServer := NewChatServer(logger)
	go func() {
		err = chatServer.Start()
		if err != nil {
			log.Fatalf("error creating chatServer: %v", err)
		}
	}()
	defer chatServer.Stop()

	// api chatServer
	apiServer := NewApiServer(chatServer)
	go func() {
		println("starting api chatServer")
		err := apiServer.Start()
		if err != nil {
			log.Fatalf("error creating api chatServer: %v", err)
		}
	}()
	defer apiServer.Stop()

	// listen for ctrl+c signal from terminal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(fmt.Sprint(<-ch))
	log.Println("Stopping API chatServer.")
}
