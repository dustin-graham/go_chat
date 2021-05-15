package main

import (
	"fmt"
	"time"
)

type Message struct {
	Client    ChatClient
	UtteredAt time.Time
	Text      string
}

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s (%s)", m.Client.Name, m.Text, m.UtteredAt.Format(time.RFC3339))
}
