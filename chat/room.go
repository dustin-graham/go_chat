package main

import "github.com/google/uuid"

type RoomId uuid.UUID

type Room struct {
	Id       RoomId
	Name     string
	Messages []Message
}

func NewRoom(name string) *Room {
	return &Room{Id: RoomId(uuid.New()), Name: name}
}

func (r Room) String() string {
	return r.Name
}
