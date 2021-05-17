package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	HelpCommand        = "//help"
	ListRoomsCommand   = "//rooms"
	JoinRoomCommand    = "//join"
	LeaveRoomCommand   = "//leave"
	CreateRoomCommand  = "//create-room"
	ListMembersCommand = "//members"
	SetNameCommand     = "//set-name"
)

type ChatServer struct {
	listener net.Listener
	clients  []*ChatClient
	lobby    Room
	rooms    map[RoomId]*Room
	logger   *log.Logger
}

func NewChatServer(logger *log.Logger) *ChatServer {
	lobby := NewRoom("Lobby")
	return &ChatServer{
		clients: []*ChatClient{},
		rooms: map[RoomId]*Room{
			lobby.Id: lobby,
		},
		lobby:  *lobby,
		logger: logger,
	}
}

func (s *ChatServer) Start() error {
	IP := os.Getenv("CHAT_SERVER_IP")
	if IP == "" {
		fmt.Println("CHAT_SERVER_IP not specified. using default 127.0.0.1")
		IP = "127.0.0.1"
	}
	port := os.Getenv("CHAT_SERVER_PORT")
	if port == "" {
		fmt.Println("CHAT_SERVER_PORT not specified. using default 8080")
		port = "8080"
	}
	address := fmt.Sprintf("%s:%s", IP, port)
	println(address)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	s.listener = listener

	s.listenForClientConnections()
	return nil
}

func (s *ChatServer) listenForClientConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Printf("failed to accept new connection: %v", err)
			return
		}
		go func() {
			client, err := s.buildClient(conn)
			if err != nil {
				fmt.Printf("error building the client: %v", err)
				// try and close the client connection
				_ = conn.Close()
				return
			}
			s.serveClient(client)
		}()
	}
}

func (s *ChatServer) buildClient(conn net.Conn) (*ChatClient, error) {
	clientName, err := getTextInput(conn, "Hello there! Welcome to the best chat service ever. Please provide your name")
	if err != nil {
		if errors.Is(err, io.EOF) {
			fmt.Println("connection closed before we could create the client")
		} else {
			fmt.Printf("error reading message input: %v", err)
		}
		return nil, err
	}
	return NewChatClient(clientName, conn, s.lobby.Id), nil
}

func (s *ChatServer) serveClient(client *ChatClient) {
	s.clients = append(s.clients, client)
	err := client.Greet(s.lobby)
	if err != nil {
		fmt.Printf("got an error while being friendly: %v", err)
	}
	for {
		message, err := client.ReadMessage("")
		if err != nil {
			if errors.Is(err, io.EOF) {
				println("closing client connection")
				// close the connection and remove the client
				s.removeClient(client)
				_ = client.Close()
				break
			} else {
				fmt.Printf("error reading message input: %v", err)
			}
		} else {
			s.processMessage(*message)
		}
	}
}

func (s *ChatServer) processMessage(message Message) {
	if message.Text == "" {
		// ignore empty messages
		return
	}
	switch message.Text {
	case HelpCommand:
		s.helpClient(message.Client)
	case ListRoomsCommand:
		roomNames := s.GetRoomNames()
		rooms := strings.Join(roomNames, ", ")
		message.Client.Notify(rooms)
	case JoinRoomCommand:
		s.joinRoom(message.Client)
	case LeaveRoomCommand:
		s.removeClientFromRoom(message.Client)
	case CreateRoomCommand:
		s.createRoom(message.Client)
	case ListMembersCommand:
		s.getRoomMembers(message.Client)
	case SetNameCommand:
		s.setClientName(message.Client)
	default:
		// just a message
		s.notifyClientsWithinRoom(message)
	}
}

func (s *ChatServer) setClientName(client *ChatClient) {
	changeNameMessage, err := client.ReadMessage("Enter the moniker by which you would like to be known:")
	if err != nil {
		fmt.Printf("error getting the client's new name")
		return
	}
	previousName := client.Name
	newName := changeNameMessage.Text
	err = client.SetName(newName)
	if err != nil {
		client.Notify(err.Error())
		return
	}
	changeNameMessage.Client.Notify(fmt.Sprintf("You got it. You shall henceforth be known as '%s'. I'll let everyone else know.", newName))
	s.notifyClientsWithinRoom(Message{
		Client:    changeNameMessage.Client,
		UtteredAt: changeNameMessage.UtteredAt,
		Text:      fmt.Sprintf("%s changed their name to %s", previousName, newName),
	})
}

func (s *ChatServer) getRoomMembers(client *ChatClient) {
	roomId := client.roomId
	roomClients := make([]string, 0)
	for _, client := range s.clients {
		if client.roomId == roomId {
			roomClients = append(roomClients, client.Name)
		}
	}
	client.Notify(fmt.Sprintf("Room members: %s", strings.Join(roomClients, ", ")))
}

func (s *ChatServer) createRoom(client *ChatClient) {
	roomNameMessage, err := client.ReadMessage("Enter the room name you would like to create:")
	if err != nil {
		fmt.Printf("error getting room name to create")
		return
	}
	roomName := roomNameMessage.Text
	for _, room := range s.rooms {
		if room.Name == roomName {
			client.Notify(fmt.Sprintf("room with name '%s' already exists", roomName))
		}
	}
	room := NewRoom(roomName)
	s.rooms[room.Id] = room
	client.SetRoomId(room.Id)
}

func (s *ChatServer) removeClientFromRoom(client *ChatClient) {
	if client.roomId == s.lobby.Id {
		client.Notify("you can checkout any time you like but you can never leave the lobby")
		return
	}
	client.roomId = s.lobby.Id
	client.Notify(fmt.Sprintf("you are now in the lobby"))
}

func (s *ChatServer) findRoom(roomName string) (*Room, error) {
	for _, room := range s.rooms {
		if room.Name == roomName {
			return room, nil
		}
	}
	return nil, ErrRoomNotFound
}

func (s *ChatServer) joinRoom(client *ChatClient) {
	roomNameMessage, err := client.ReadMessage("Enter the room name you would like to join:")
	if err != nil {
		fmt.Printf("error getting room name to join")
		return
	}
	room, err := s.findRoom(roomNameMessage.Text)
	if err != nil {
		if errors.Is(err, ErrRoomNotFound) {
			client.Notify(fmt.Sprintf("room not found: %s", roomNameMessage.Text))
		} else {
			client.Notify(fmt.Sprintf("error adding you to room: %s", roomNameMessage.Text))
		}
		return
	}
	client.SetRoomId(room.Id)
	client.Notify(fmt.Sprintf("You have now joined '%s'", room.Name))
}

func (s *ChatServer) GetRoomNames() []string {
	roomNames := make([]string, 0)
	for _, room := range s.rooms {
		roomNames = append(roomNames, room.Name)
	}
	return roomNames
}

func (s *ChatServer) helpClient(client *ChatClient) {
	err := client.SendHelp()
	if err != nil {
		fmt.Printf("encountered an error while being helpful: %v", err)
	}
}

func (s *ChatServer) logMessage(message Message) {
	room := s.rooms[message.Client.roomId]
	s.logger.Printf("In %s: %s", room.Name, message.String())
}

func (s *ChatServer) notifyClientsWithinRoom(message Message) {
	s.logMessage(message)
	roomId := message.Client.roomId

	// record this message in the room
	s.rooms[roomId].Messages = append(s.rooms[roomId].Messages, message)

	for _, client := range s.clients {
		if client.roomId == roomId && client.ClientId != message.Client.ClientId {
			client.Notify(message.String())
		}
	}
}

func (s *ChatServer) GetRoomMessages(roomName string) ([]Message, error) {
	room, err := s.findRoom(roomName)
	if err != nil {
		return []Message{}, err
	}
	return room.Messages, nil
}

func (s *ChatServer) PostMessageToRoom(roomName string, messageText string) error {
	room, err := s.findRoom(roomName)
	if err != nil {
		return err
	}
	s.notifyClientsWithinRoom(Message{
		Client: &ChatClient{
			Name:   "API User",
			roomId: room.Id,
		},
		UtteredAt: time.Now(),
		Text:      messageText,
	})
	return nil
}

func (s *ChatServer) removeClient(client *ChatClient) {
	var clientIndex int
	for i, chatClient := range s.clients {
		if chatClient == client {
			clientIndex = i
		}
	}
	s.clients = append(s.clients[:clientIndex], s.clients[clientIndex+1:]...)
}

func (s *ChatServer) Stop() {
	if err := s.listener.Close(); err != nil {
		log.Fatalf("Could not shut down server correctly: %v\n", err)
	}
}
