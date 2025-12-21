// server
package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

// Client represents a connected chat client
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// Map to track all connected clients
var (
	clients = make(map[*Client]bool)
	mu      sync.Mutex
)

// Broadcast a message to all clients except the sender
func broadcast(sender *Client, message string) {
	mu.Lock()
	defer mu.Unlock()

	for client := range clients {
		// Skip the sender
		if client == sender {
			continue
		}

		_, err := client.writer.WriteString(message)
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

		clientAddr := client.conn.RemoteAddr().String()
		fmt.Printf("server: received from %s: %s", clientAddr, message)

		// Format the broadcast message with the sender's address
		broadcastMsg := fmt.Sprintf("From %s: %s", clientAddr, message)

		// Broadcast to all other clients
		broadcast(client, broadcastMsg)

		// Also send a confirmation to the sender
		client.writer.WriteString("Message sent to all clients\n")
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
		}

		// Add client to tracking map
		mu.Lock()
		clients[client] = true
		mu.Unlock()

		fmt.Println("server: client connected from", client.conn.RemoteAddr())
		fmt.Printf("server: %d clients connected\n", len(clients))

		go handleClient(client)
	}
}
