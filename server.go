// server
package main

import (
	"bufio"
	"fmt"
	"lancom/protocol"
	"net"
	"sync"
	"sync/atomic"
)

// Client represents a connected chat client
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	id     string
}

// Map to track all connected clients
var (
	clients         = make(map[*Client]bool)
	mu              sync.Mutex
	clientIDCounter uint64 // atomic counter for sequential IDs
)

// getNextClientID generates a sequential, unique client ID
// Uses atomic operations to ensure thread-safety and no duplicates
func getNextClientID() string {
	id := atomic.AddUint64(&clientIDCounter, 1)
	return fmt.Sprintf("client-%d", id)
}

// Broadcast a message to all clients except the sender
func broadcast(sender *Client, message *protocol.Message) {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		// Skip the sender
		if client == sender {
			continue
		}

		msgJson, err := protocol.Encode(message)
		if err != nil {
			fmt.Println("server: failed to parse Message object:", err)
		}

		_, err = client.writer.WriteString(string(msgJson) + "\n")
		if err != nil {
			// If we can't write to a client, assume it's disconnected
			// We'll remove it next time it tries to read/writes
			continue
		}
		client.writer.Flush()
	}
}

func handleClient(client *Client) {
	defer func() {
		// Clean up when client disconnects
		mu.Lock()
		delete(clients, client)
		mu.Unlock()

		fmt.Println("server: client disconnected:", client.conn.RemoteAddr())
		client.conn.Close()
	}()

	// Read lines and broadcast them
	for {
		message, err := client.reader.ReadString('\n')
		if err != nil {
			return // Exit and trigger the deferred cleanup
		}

		// Prepare Message object from received json
		msgObj, err := protocol.Decode([]byte(message))
		if err != nil {
			fmt.Println("server: decode error:", err)
		}

		// Broadcast to all other clients
		broadcast(client, msgObj)

		// Also send a confirmation to the sender
		msgAck := protocol.Message{
			Type: protocol.TypeChatAck,
			From: "server",
			To:   "client",
			Body: "Message sent to all clients",
		}

		data, err := protocol.Encode(&msgAck)
		if err != nil {
			fmt.Println("server: encode error: ", err)
		}
		client.writer.WriteString(string(data) + "\n")
		client.writer.Flush()
	}
}

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
			id:     getNextClientID(),
		}

		// Add client to tracking map
		mu.Lock()
		clients[client] = true
		mu.Unlock()

		fmt.Printf("server: client connected from %s (ID: %s)\n", client.conn.RemoteAddr(), client.id)
		fmt.Printf("server: %d clients connected\n", len(clients))

		go handleClient(client)
	}
}
