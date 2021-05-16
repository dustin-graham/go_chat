package main

import "fmt"

type ChatError struct {
	Err     error
	Message string
}

func (c *ChatError) Error() string {
	return fmt.Sprintf("%s: %s", c.Message, c.Err.Error())
}

var (
	ErrRoomNotFound = &ChatError{
		Err:     nil,
		Message: "room not found",
	}
)
