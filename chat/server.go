package main

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"os"
	"strings"
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
	rooms    map[uuid.UUID]*Room
	logger   *log.Logger
}

func NewChatServer(logger *log.Logger) *ChatServer {
	lobby := NewRoom(uuid.New(), "Lobby")
	return &ChatServer{
		clients: []*ChatClient{},
		rooms: map[uuid.UUID]*Room{
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
			client, err := s.BuildClient(conn)
			if err != nil {
				fmt.Printf("error building the client: %v", err)
				// try and close the client connection
				_ = conn.Close()
				return
			}
			s.ServeClient(client)
		}()
	}
}

func (s *ChatServer) BuildClient(conn net.Conn) (*ChatClient, error) {
	clientName, err := getTextInput(conn, "Hello there! Welcome to the best chat service ever. Please provide your name")
	if err != nil {
		if errors.Is(err, io.EOF) {
			fmt.Println("connection closed before we could create the client")
		} else {
			fmt.Printf("error reading message input: %v", err)
		}
		return nil, err
	}
	return NewChatClient(uuid.New(), clientName, conn, s.lobby.Id), nil
}

func (s *ChatServer) ServeClient(client *ChatClient) {
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
				s.RemoveClient(client)
				_ = client.Close()
				break
			} else {
				fmt.Printf("error reading message input: %v", err)
			}
		} else {
			s.ProcessMessage(*message)
		}
	}
}

func (s *ChatServer) ProcessMessage(message Message) {
	if message.Text == "" {
		// ignore empty messages
		return
	}
	switch message.Text {
	case HelpCommand:
		s.helpClient(message.Client)
	case ListRoomsCommand:
		s.listRooms(message.Client)
	case JoinRoomCommand:
		s.joinRoom(message.Client)
	case LeaveRoomCommand:
		s.removeClientFromRoom(message.Client)
	case CreateRoomCommand:
		s.createRoom(message.Client)
	case ListMembersCommand:
		s.listMembers(message.Client)
	case SetNameCommand:
		s.setClientName(message.Client)
	default:
		// just a message
		err := s.NotifyClientsWithinRoom(message)
		if err != nil {
			s.logger.Printf("error sending message to room: %v", err)
		}
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
		_ = client.Notify(err.Error())
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
	}
}

func (s *ChatServer) listMembers(client *ChatClient) {
	roomId := client.RoomId
	roomClients := make([]string, 0)
	for _, client := range s.clients {
		if client.RoomId == roomId {
			roomClients = append(roomClients, client.Name)
		}
	}
	err := client.Notify(fmt.Sprintf("Room members: %s", strings.Join(roomClients, ", ")))
	if err != nil {
		fmt.Printf("error responding to room member request: %v", err)
	}
}

func (s *ChatServer) createRoom(client *ChatClient) {
	roomName, err := client.ReadMessage("Enter the room name you would like to create:")
	if err != nil {
		fmt.Printf("error getting room name to create")
		return
	}
	room, err := s.CreateRoom(roomName.Text)
	if err != nil {
		_ = client.Notify(err.Error())
		return
	}
	err = client.JoinRoom(room)
	if err != nil {
		_ = client.Notify(err.Error())
		return
	}
}

func (s *ChatServer) removeClientFromRoom(client *ChatClient) {
	if client.RoomId == s.lobby.Id {
		err := client.Notify("you can checkout any time you like but you can never leave the lobby")
		if err != nil {
			fmt.Printf("error responding to bad //leave request: %v", err)
		}
		return
	}
	client.RoomId = s.lobby.Id
	err := client.Notify(fmt.Sprintf("you are now in the lobby"))
	if err != nil {
		fmt.Printf("error returning to lobby: %v", err)
	}
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
			_ = writeText(client.conn, fmt.Sprintf("room not found: %s", roomNameMessage.Text))
		} else {
			_ = writeText(client.conn, fmt.Sprintf("error adding you to room: %s", roomNameMessage.Text))
		}
		return
	}
	err = client.JoinRoom(room)
	if err != nil {
		err := client.Notify(err.Error())
		if err != nil {
			fmt.Printf("error responding to join room request: %v", err)
		}
	}
	err = client.Notify(fmt.Sprintf("could not find room named %s", roomNameMessage.Text))
	if err != nil {
		fmt.Printf("error responding to create room request: %v", err)
	}
}

func (s *ChatServer) listRooms(client *ChatClient) {
	roomNames := make([]string, 0)
	for _, room := range s.rooms {
		roomNames = append(roomNames, room.Name)
	}
	rooms := strings.Join(roomNames, ", ")
	err := client.Notify(rooms)
	if err != nil {
		fmt.Printf("error sending rooms list to %s: %v", client.Name, err)
	}
}

func (s *ChatServer) helpClient(client *ChatClient) {
	err := client.SendHelp()
	if err != nil {
		fmt.Printf("encountered an error while being helpful: %v", err)
	}
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

func (s *ChatServer) logMessage(message Message) {
	room := s.rooms[message.Client.RoomId]
	s.logger.Printf("In %s: %s", room.Name, message.String())
}

func (s *ChatServer) NotifyClientsWithinRoom(message Message) error {
	s.logMessage(message)
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
