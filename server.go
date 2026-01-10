// server
package main

import (
	"bufio"
	"errors"
	"fmt"
	"lancom/protocol"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

// Map to track all connected clients
var (
	clients         = make(map[*Client]bool)
	mu              sync.Mutex
	clientIDCounter uint64 // atomic counter for sequential IDs
)

// Client represents a connected chat client
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	id     string
	joined bool
}

// getNextClientID generates a sequential, unique client ID
// Uses atomic operations to ensure thread-safety and no duplicates
func getNextClientID() string {
	id := atomic.AddUint64(&clientIDCounter, 1)
	return fmt.Sprintf("client-%d", id)
}

// msgWriter: writes message to the connection
func msgWriter(m *protocol.Message, client *Client) error {
	m.Version = protocol.Version
	data, err := protocol.Encode(m)
	if err != nil {
		return err
	}
	_, err = client.writer.WriteString(string(data) + "\n")
	if err != nil {
		return err
	}
	client.writer.Flush()
	return nil
}

// Broadcaster: broadcast a message to all clients except the sender
func broadcaster(msg *protocol.Message, sender *Client) {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		// Skip the sender
		if client == sender {
			continue
		}
		err := msgWriter(msg, client)
		if err != nil {
			continue
		}
	}
}

// Join handler: handles what to do on join
func joinHandler(client *Client) error {
	mu.Lock()

	if client.joined {
		return fmt.Errorf("Client already joined")
	}
	clients[client] = true
	client.id = getNextClientID()
	client.joined = true
	mu.Unlock()

	msg := protocol.Message{
		Type: protocol.TypeJoinAck,
		From: protocol.Server,
		To:   "client",
		Body: client.id,
	}

	return msgWriter(&msg, client)
}

// Chat handler: handles what do on chat request
func chatHandler(msg *protocol.Message, client *Client) error {
	msg.From = client.id
	broadcaster(msg, client)
	msgAck := protocol.Message{
		Type: protocol.TypeChatAck,
		From: protocol.Server,
		To:   client.id,
		Body: "Message sent to all...",
	}

	err := msgWriter(&msgAck, client)
	if err != nil {
		return err
	}

	return nil
}

// Leave handler: handles gracefull shutdown or delete of client on leave
func leaveHandler(client *Client) error {
	mu.Lock()
	if _, exits := clients[client]; exits && client.joined {
		delete(clients, client)
		client.joined = false
		mu.Unlock()
		client.conn.Close()
		return nil
	}
	mu.Unlock()
	return nil
}

// Sementic Validator: server side validator
func semanticValidator(m *protocol.Message, client *Client) error {
	if m.Type == protocol.TypeJoinAck && m.From != protocol.Server {
		return fmt.Errorf("acknowledgement can only be sent by server")
	}
	if m.Type == protocol.TypeJoinAck {
		return fmt.Errorf("client can't sent %s (join acknowledgements)", protocol.TypeJoinAck)
	}
	if m.Type == protocol.TypeLeave && !client.joined {
		return fmt.Errorf("client (%s) is not joined", client.id)
	}
	
	return nil
}

// Message handler: decides what to do for different types of message
func messageHandler(msg *string, client *Client) error {
	msgObj, err := protocol.Decode([]byte(*msg))
	if err != nil {
		return err
	}
	err = msgObj.Validate()
	if err != nil {
		return err
	}

	if !client.joined && msgObj.Type != protocol.TypeJoin {
		return errors.New("client must join first")
	}

	err = semanticValidator(msgObj, client)
	if err != nil {
		return err
	}
	switch msgObj.Type {
	case protocol.TypeJoin:
		return joinHandler(client)
	case protocol.TypeChat:
		return chatHandler(msgObj, client)
	case protocol.TypeLeave:
		return leaveHandler(client)
	}
	return nil
}

// Handles all the functionality for a client like reading and writing
// Handles all the message types like `join`, `chat`...
func clientHandler(client *Client) {
	defer leaveHandler(client)

	// Read lines and broadcast them
	for {
		msg, err := client.reader.ReadString('\n')
		if err != nil {
			return // Exit and trigger the deferred cleanup
		}

		msg = strings.TrimSuffix(msg, "\n")

		err = messageHandler(&msg, client)
		if err != nil {
			fmt.Println("error processing message from %s: %v\n", client.id, err)
			errMsg := protocol.Message{
				Type: protocol.TypeError,
				From: protocol.Server,
				To:   client.id,
				Body: string(err.Error()),
			}
			msgWriter(&errMsg, client)
		}
	}
}

// Main entry for the client-to-server connection
func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("server: failed to start")
		panic(err)
	}
	defer ln.Close()

	fmt.Println("server: listening on 127.0.0.1:9000")

	// Accept clients in a loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("server: accept error:", err)
			continue
		}

		// Create a client from the connection
		client := &Client{
			conn:   conn,
			reader: bufio.NewReader(conn),
			writer: bufio.NewWriter(conn),
		}

		go clientHandler(client)
	}
}
