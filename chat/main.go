package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logFilePath := os.Getenv("CHAT_SERVER_LOG_FILE_PATH")
	if logFilePath == "" {
		println("log file path not specified, using default")
		logFilePath = "chat_log.txt"
	}
	logFile, err := os.OpenFile(
		logFilePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		// logging is an important part of the app, we've got to bail if we can't do logging
		log.Fatalf("failed to open log file")
	}

	// chat server
	logger := log.New(logFile, "", log.LstdFlags)
	chatServer := NewChatServer(logger)
	go func() {
		err = chatServer.Start()
		if err != nil {
			log.Fatalf("error creating server: %v", err)
		}
	}()
	defer chatServer.Stop()

	// api server
	apiServer := NewApiServer(chatServer)
	go func() {
		println("starting api server")
		err := apiServer.Start()
		if err != nil {
			log.Fatalf("error creating api server: %v", err)
		}
	}()
	defer apiServer.Stop()

	// listen for ctrl+c signal from terminal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(fmt.Sprint(<-ch))
	log.Println("Stopping API server.")
}
