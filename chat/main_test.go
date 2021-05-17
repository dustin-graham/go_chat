package main

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestChatServer_RemoveClient(t *testing.T) {
	susan := NewChatClient("Susan", nil, RoomId(uuid.New()))
	dave := NewChatClient("Dave", nil, RoomId(uuid.New()))
	bill := NewChatClient("Bill", nil, RoomId(uuid.New()))
	type args struct {
		clientToRemove  *ChatClient
		expectedClients []*ChatClient
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "remove 0",
			args: args{
				clientToRemove:  susan,
				expectedClients: []*ChatClient{dave, bill},
			},
		},
		{
			name: "remove 1",
			args: args{
				clientToRemove:  dave,
				expectedClients: []*ChatClient{susan, bill},
			},
		},
		{
			name: "remove 2",
			args: args{
				clientToRemove:  bill,
				expectedClients: []*ChatClient{susan, dave},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ChatServer{
				clients: []*ChatClient{
					susan,
					dave,
					bill,
				},
			}
			s.removeClient(tt.args.clientToRemove)
			equal := reflect.DeepEqual(tt.args.expectedClients, s.clients)
			assert.EqualValues(t, true, equal)
		})
	}
}
