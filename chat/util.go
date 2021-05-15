package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

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
