package main

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"strings"
)

type ChatServer struct {
	listener net.Listener
	clients  []*ChatClient
	lobby    Room
	rooms    map[uuid.UUID]*Room
}

func NewChatServer() *ChatServer {
	lobby := NewRoom(uuid.New(), "Lobby")
	return &ChatServer{
		clients: []*ChatClient{},
		rooms: map[uuid.UUID]*Room{
			lobby.Id: lobby,
		},
		lobby: *lobby,
	}
}

func (s *ChatServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		return err
	}
	s.listener = listener

	err = s.listenForClientConnections()
	if err != nil {
		log.Fatalf("failed to listen to server: %v", err)
	}
	return nil
}

func (s *ChatServer) listenForClientConnections() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Fatalf("error accepting connection: %v", err)
		}
		go func() {
			client, err := s.BuildClient(conn)
			if err != nil {
				log.Fatalf("error building the client: %v", err)
			}
			go s.ServeClient(client)
		}()
	}
}

func (s *ChatServer) BuildClient(conn net.Conn) (*ChatClient, error) {
	clientName, err := getTextInput(conn, "Hello there! Welcome to the best chat service ever. Please provide your name")
	if err != nil {
		return nil, err
	}
	return NewChatClient(uuid.New(), clientName, conn, s.lobby.Id), nil
}

func (s *ChatServer) ServeClient(client *ChatClient) {
	s.clients = append(s.clients, client)
	client.Greet(s.lobby)
	for {
		message, err := client.ReadMessage("")
		if err != nil {
			if errors.Is(err, io.EOF) {
				println("closing client connection")
				// close the connection and remove the client
				s.RemoveClient(client)
				_ = client.Close()
				break
			} else {
				log.Fatalf("error reading message input")
			}
		}
		s.ProcessMessage(*message)
	}
}

func (s *ChatServer) ProcessMessage(message Message) {
	messageRoomId := message.Client.RoomId
	if message.Text == "//help" {
		message.Client.SendHelp()
	} else if message.Text == "//rooms" {
		roomNames := make([]string, 0)
		for _, room := range s.rooms {
			roomNames = append(roomNames, room.Name)
		}
		rooms := strings.Join(roomNames, ", ")
		err := message.Client.Notify(rooms)
		if err != nil {
			fmt.Printf("error sending rooms list to %s: %v", message.Client.Name, err)
		}
	} else if message.Text == "//join" {
		roomName, err := message.Client.ReadMessage("Enter the room name you would like to join:")
		if err != nil {
			fmt.Printf("error getting room name to join")
		} else {
			for _, room := range s.rooms {
				if room.Name == roomName.Text {
					err := message.Client.JoinRoom(room)
					if err != nil {
						err := message.Client.Notify(err.Error())
						if err != nil {
							fmt.Printf("error responding to join room request: %v", err)
						}
					}
					return
				}
			}
			err := message.Client.Notify(fmt.Sprintf("could not find room named %s", roomName.Text))
			if err != nil {
				fmt.Printf("error responding to create room request: %v", err)
			}
		}
	} else if message.Text == "//leave" {
		if message.Client.RoomId == s.lobby.Id {
			err := message.Client.Notify("you can checkout any time you like but you can never leave the lobby")
			if err != nil {
				fmt.Printf("error responding to bad //leave request: %v", err)
			}
			return
		}
		err := s.RemoveClientToLobby(&message.Client)
		if err != nil {
			fmt.Printf("error returning to lobby: %v", err)
		}
	} else if message.Text == "//create-room" {
		roomName, err := message.Client.ReadMessage("Enter the room name you would like to create:")
		if err != nil {
			fmt.Printf("error getting room name to create")
		} else {
			room, err := s.CreateRoom(roomName.Text)
			if err != nil {
				err := message.Client.Notify(err.Error())
				if err != nil {
					fmt.Printf("error responding to create room request: %v", err)
				}
				return
			}
			err = message.Client.JoinRoom(room)
			if err != nil {
				err := message.Client.Notify(err.Error())
				if err != nil {
					fmt.Printf("error notifying client of joining room: %v", err)
				}
				return
			}
		}
	} else if message.Text == "//members" {
		roomId := messageRoomId
		roomClients := make([]string, 0)
		for _, client := range s.clients {
			if client.RoomId == roomId {
				roomClients = append(roomClients, client.Name)
			}
		}
		err := message.Client.Notify(fmt.Sprintf("Room members: %s", strings.Join(roomClients, ", ")))
		if err != nil {
			fmt.Printf("error responding to room member request: %v", err)
		}
	} else if message.Text == "//set-name" {
		changeNameMessage, err := message.Client.ReadMessage("Enter the moniker by which you would like to be known:")
		if err != nil {
			fmt.Printf("error getting the client's new name")
		} else {
			previousName := message.Client.Name
			newName := changeNameMessage.Text
			err := message.Client.SetName(newName)
			if err != nil {
				err := message.Client.Notify(err.Error())
				if err != nil {
					fmt.Printf("error responding to set-name request: %v", err)
				}
				return
			}
			err = changeNameMessage.Client.Notify(fmt.Sprintf("You got it. You shall henceforth be known as '%s'. I'll let everyone else know.", newName))
			if err != nil {
				fmt.Printf("error confirming name change with requesting client: %v", err)
			}
			err = s.NotifyClientsWithinRoom(Message{
				Client:    changeNameMessage.Client,
				UtteredAt: changeNameMessage.UtteredAt,
				Text:      fmt.Sprintf("%s changed their name to %s", previousName, newName),
			})
			if err != nil {
				fmt.Printf("error notifying room members about member name change: %v", err)
				return
			}
		}
	} else {
		// just a message
		err := s.NotifyClientsWithinRoom(message)
		if err != nil {
			fmt.Printf("error notifying clients of new message")
		}
	}
}

func (s *ChatServer) RemoveClientToLobby(client *ChatClient) error {
	client.RoomId = s.lobby.Id
	return client.Notify(fmt.Sprintf("you are now in the lobby"))
}

func (s *ChatServer) CreateRoom(roomName string) (*Room, error) {
	for _, room := range s.rooms {
		if room.Name == roomName {
			return nil, fmt.Errorf("room with name '%s' already exists", roomName)
		}
	}
	roomId := uuid.New()
	room := NewRoom(roomId, roomName)
	s.rooms[roomId] = room
	return room, nil
}

func (s *ChatServer) NotifyClientsWithinRoom(message Message) error {
	for _, client := range s.clients {
		if client.RoomId == message.Client.RoomId && client.ClientId != message.Client.ClientId {
			return client.Notify(message.String())
		}
	}
	return nil
}

func (s *ChatServer) RemoveClient(client *ChatClient) {
	var clientIndex int
	for i, chatClient := range s.clients {
		if chatClient == client {
			clientIndex = i
		}
	}
	s.clients = append(s.clients[:clientIndex], s.clients[clientIndex+1:]...)
}

func (s *ChatServer) Stop() error {
	if err := s.listener.Close(); err != nil {
		return err
	}
	return nil
}
