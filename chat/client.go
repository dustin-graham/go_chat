package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"net"
	"time"
)

type ClientId uuid.UUID
type ChatClient struct {
	ClientId ClientId
	Name     string
	conn     net.Conn
	roomId   RoomId
	reader   *bufio.Reader
}

func NewChatClient(name string, conn net.Conn, roomId RoomId) *ChatClient {
	reader := bufio.NewReader(conn)
	return &ChatClient{
		ClientId: ClientId(uuid.New()),
		Name:     name,
		conn:     conn,
		roomId:   roomId,
		reader:   reader,
	}
}

func (c *ChatClient) ReadMessage(prompt string) (*Message, error) {
	if prompt != "" {
		_, err := c.conn.Write([]byte(fmt.Sprintf("%s\n", prompt)))
		if err != nil {
			return nil, err
		}
	}
	input, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	input = trimMessage(input)
	return &Message{
		Client:    c,
		UtteredAt: time.Now(),
		Text:      input,
	}, nil
}

func (c *ChatClient) Notify(message string) {
	if err := writeText(c.conn, message); err != nil {
		fmt.Printf("error sending '%s' to %s", message, c.Name)
	}
}

func (c *ChatClient) Greet(room Room) error {
	err := writeText(c.conn, fmt.Sprintf("Hello %s. You are now in the %s room. Feel free to speak your mind. Type //help if you need a hand", c.Name, room.Name))
	if err != nil {
		return err
	}
	return nil
}

func (c *ChatClient) SendHelp() error {
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
		return err
	}
	return nil
}

func (c *ChatClient) SetRoomId(roomId RoomId) {
	c.roomId = roomId
}

func (c *ChatClient) GetRoomId() RoomId {
	return c.roomId
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
