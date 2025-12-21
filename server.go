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

func handleClient(client *Client) {
	defer func() {
		// Clean up when client disconnects
		mu.Lock()
		delete(clients, client)
		mu.Unlock()

		fmt.Println("server: client disconnected:", client.conn.RemoteAddr())
		client.conn.Close()
	}()

	// Read lines and echo them back
	for {
		message, err := client.reader.ReadString('\n')
		if err != nil {
			return // Exit and trigger the deferred cleanup
		}

		fmt.Printf("server: received from %s: %s", client.conn.RemoteAddr(), message)

		// Just echo for now, we'll implement broadcast next
		_, err = client.writer.WriteString("Echo: " + message)
		if err != nil {
			return
		}

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
