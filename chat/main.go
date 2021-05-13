package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"strings"
	"time"
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

type Room struct {
	Id   uuid.UUID
	Name string
}

func NewRoom(id uuid.UUID, name string) *Room {
	return &Room{Id: id, Name: name}
}

func (r Room) String() string {
	return r.Name
}

type ChatClient struct {
	ClientId uuid.UUID
	Name     string
	conn     net.Conn
	RoomId   uuid.UUID
}

type Message struct {
	Client    ChatClient
	UtteredAt time.Time
	Text      string
}

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s (%s)", m.Client.Name, m.Text, m.UtteredAt.Format(time.RFC3339))
}

func main() {
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

	err = server.Listen()
	if err != nil {
		log.Fatalf("failed to listen to server: %v", err)
	}
}

func (s *ChatServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

func (s *ChatServer) Stop() error {
	if err := s.listener.Close(); err != nil {
		return err
	}
	return nil
}

func (s *ChatServer) Listen() error {
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
			go s.Serve(client)
		}()
	}
}

func (s *ChatServer) BuildClient(conn net.Conn) (*ChatClient, error) {
	clientName, err := getTextInput(conn, "Hello there! Welcome to the best chat service ever. Please provide your name")
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		ClientId: uuid.New(),
		Name:     clientName,
		conn:     conn,
		RoomId:   s.lobby.Id,
	}, nil
}

func (s *ChatServer) Serve(client *ChatClient) {
	s.clients = append(s.clients, client)
	client.Greet(s.lobby)
Serve:
	for {
		message, err := client.ReadMessage("")
		if err != nil {
			if errors.Is(err, io.EOF) {
				println("closing client connection")
				// close the connection and remove the client
				s.RemoveClient(client)
				_ = client.Close()
				break Serve
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
	room := &Room{
		Id:   roomId,
		Name: roomName,
	}
	s.rooms[roomId] = room
	return room, nil
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

func (s *ChatServer) NotifyClientsWithinRoom(message Message) error {
	for _, client := range s.clients {
		if client.RoomId == message.Client.RoomId && client.ClientId != message.Client.ClientId {
			return client.Notify(message.String())
		}
	}
	return nil
}

func (c *ChatClient) Greet(room Room) {
	err := writeText(c.conn, fmt.Sprintf("Hello %s. You are now in the %s room. Feel free to speak your mind. Type //help if you need a hand", c.Name, room.Name))
	if err != nil {
		log.Fatalf("failed to greet the client: %v", err)
	}
}

func (c *ChatClient) JoinRoom(room *Room) error {
	c.RoomId = room.Id
	return c.Notify(fmt.Sprintf("You have now joined '%s'", room.Name))
}

func (c *ChatClient) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be blank")
	}
	c.Name = name
	return nil
}

func (c *ChatClient) SendHelp() {
	help := `
Use one of the following commands and your wildest dreams will come true

//rooms - list the available rooms to join
//join - join a room
//leave - leave the room you are in. if you are not in a room it does nothing... or does it?
//create-room - create a room and joins a room
//members - lists the members of the room you are in
//set-name - change your name
//help - get help... but you knew that already, didn't you?
`
	err := writeText(c.conn, help)
	if err != nil {
		log.Fatalf("failed to be helpful: %v", err)
	}
}

func (c *ChatClient) Notify(message string) error {
	if err := writeText(c.conn, message); err != nil {
		return err
	}
	return nil
}

func (c *ChatClient) ReadMessage(prompt string) (*Message, error) {
	if prompt != "" {
		_, err := c.conn.Write([]byte(fmt.Sprintf("%s\n", prompt)))
		if err != nil {
			return nil, err
		}
	}
	input, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return nil, err
	}
	input = trimMessage(input)
	return &Message{
		Client:    *c,
		UtteredAt: time.Now(),
		Text:      input,
	}, nil
}

func (c *ChatClient) Close() error {
	return c.conn.Close()
}

func trimMessage(messageText string) string {
	return strings.TrimSuffix(messageText, "\n")
}

func writeText(conn net.Conn, text string) error {
	if _, err := conn.Write([]byte(fmt.Sprintf("%s\n", text))); err != nil {
		return err
	}
	return nil
}

func getTextInput(conn net.Conn, prompt string) (string, error) {
	_, err := conn.Write([]byte(fmt.Sprintf("%s\n", prompt)))
	if err != nil {
		return "", err
	}
	input, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	input = trimMessage(input)
	return input, nil
}
