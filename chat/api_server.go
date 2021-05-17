package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type ChatTextMessage struct {
	Message   string `json:"message"`
	Author    string `json:"author"`
	UtteredAt string `json:"utteredAt"`
}

type ChatTextMessageList struct {
	Messages []ChatTextMessage `json:"messages"`
}

type ApiServer struct {
	server     *http.Server
	chatServer *ChatServer
}

func NewApiServer(chatServer *ChatServer) *ApiServer {
	return &ApiServer{
		chatServer: chatServer,
	}
}

func (s *ApiServer) Start() error {
	if s.server != nil {
		return fmt.Errorf("server already started")
	}
	s.server = s.buildServer()
	listener, err := net.Listen("tcp", s.getAddress())
	if err != nil {
		log.Fatalf("Error occurred: %s", err.Error())
	}
	if err := s.server.Serve(listener); err != nil {
		return err
	}
	return nil
}

func (s *ApiServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not shut down server correctly: %v\n", err)
	}
	s.server = nil
}

func (s *ApiServer) buildServer() *http.Server {
	return &http.Server{
		Handler: s.buildRouter(),
	}
}

func (s *ApiServer) buildRouter() http.Handler {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
				roomName := r.URL.Query().Get("roomName")
				if roomName == "" {
					render.Render(w, r, ApiErrBadRequestRoomNotProvided)
					return
				}
				messages, err := s.chatServer.GetRoomMessages(roomName)
				if err != nil {
					if errors.Is(err, ErrRoomNotFound) {
						render.Render(w, r, ApiErrRoomNotFound)
					} else {
						render.Render(w, r, &ApiError{
							Err:        err,
							StatusCode: http.StatusInternalServerError,
							StatusText: "unexpected error",
							Message:    "unexpected error",
						})
					}
					return
				}
				chatMessages := make([]ChatTextMessage, len(messages))
				for i, message := range messages {
					chatMessages[i] = ChatTextMessage{
						Message:   message.Text,
						Author:    message.Client.Name,
						UtteredAt: message.UtteredAt.Format(time.RFC3339),
					}
				}
				render.JSON(w, r, ChatTextMessageList{Messages: chatMessages})
			})
			r.Post("/messages", func(w http.ResponseWriter, r *http.Request) {
				roomName := r.URL.Query().Get("roomName")
				if roomName == "" {
					render.Render(w, r, ApiErrBadRequestRoomNotProvided)
					return
				}
				message := r.URL.Query().Get("message")
				err := s.chatServer.PostMessageToRoom(roomName, message)
				if err != nil {
					if errors.Is(err, ErrRoomNotFound) {
						render.Render(w, r, ApiErrRoomNotFound)
					} else {
						render.Render(w, r, &ApiError{
							Err:        err,
							StatusCode: http.StatusInternalServerError,
							StatusText: "unexpected error",
							Message:    "unexpected error",
						})
					}
					return
				}
			})
		})
	})
	return router
}

func (s *ApiServer) getAddress() string {
	IP := os.Getenv("CHAT_SERVER_IP")
	if IP == "" {
		fmt.Println("CHAT_SERVER_IP not specified. using default 127.0.0.1")
		IP = "127.0.0.1"
	}
	port := os.Getenv("API_SERVER_PORT")
	if port == "" {
		fmt.Println("CHAT_SERVER_PORT not specified. using default 8081")
		port = "8081"
	}
	return fmt.Sprintf("%s:%s", IP, port)
}
