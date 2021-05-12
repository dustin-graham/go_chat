package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type ChatServer struct {
	listener net.Listener
	clients  []*ChatClient
}

type ChatClient struct {
	Name string
	conn net.Conn
}

type Message struct {
	ClientName string
	UtteredAt  time.Time
	Text       string
}

func (m *Message) String() string {
	return fmt.Sprintf("%s said: %s (%s)", m.ClientName, m.Text, m.UtteredAt.Format(time.RFC3339))
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

func NewChatServer() *ChatServer {
	return &ChatServer{clients: []*ChatClient{}}
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
			s.clients = append(s.clients, client)
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
		Name: clientName,
		conn: conn,
	}, nil
}

func (s *ChatServer) Serve(client *ChatClient) {
	client.Greet()
	for {
		message, err := client.ReadMessage()
		if err != nil {
			log.Fatalf("error reading message input")
		}
		s.NotifyClients(message)
	}
}

func (s *ChatServer) NotifyClients(message *Message) {
	for _, client := range s.clients {
		err := client.Notify(message)
		if err != nil {
			log.Fatalf("error notifying client")
		}
	}
}

func (c *ChatClient) Greet() {
	err := writeText(c.conn, fmt.Sprintf("Hello %s. Feel free to speak your mind", c.Name))
	if err != nil {
		log.Fatalf("failed to greet the client: %v", err)
	}
}

func (c *ChatClient) Notify(message *Message) error {
	if err := writeText(c.conn, message.String()); err != nil {
		return err
	}
	return nil
}

func (c *ChatClient) ReadMessage() (*Message, error) {
	input, err := bufio.NewReader(c.conn).ReadString('\n')
	input = trimMessage(input)
	if err != nil {
		return nil, err
	}
	return &Message{
		ClientName: c.Name,
		UtteredAt:  time.Now(),
		Text:       input,
	}, nil
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
