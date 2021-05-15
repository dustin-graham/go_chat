package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"time"
)

type ChatClient struct {
	ClientId uuid.UUID
	Name     string
	conn     net.Conn
	RoomId   uuid.UUID
}

func NewChatClient(clientId uuid.UUID, name string, conn net.Conn, roomId uuid.UUID) *ChatClient {
	return &ChatClient{ClientId: clientId, Name: name, conn: conn, RoomId: roomId}
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

func (c *ChatClient) Notify(message string) error {
	if err := writeText(c.conn, message); err != nil {
		return err
	}
	return nil
}

func (c *ChatClient) Greet(room Room) {
	err := writeText(c.conn, fmt.Sprintf("Hello %s. You are now in the %s room. Feel free to speak your mind. Type //help if you need a hand", c.Name, room.Name))
	if err != nil {
		log.Fatalf("failed to greet the client: %v", err)
	}
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

func (c *ChatClient) Close() error {
	return c.conn.Close()
}
