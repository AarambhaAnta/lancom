// server
package main

import (
	"bufio"
	"fmt"
	"net"
)

// Client represents a connected chat client
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func handleClient(client *Client) {
	// Read lines and echo them back
	for {
		message, err := client.reader.ReadString('\n')
		if err != nil {
			fmt.Println("server: read error:", err)
			return
		}

		fmt.Println("server: received: ", message)

		// Echo the message back
		_, err = client.writer.WriteString("Echo: " + message)
		if err != nil {
			fmt.Println("server: write error:", err)
			return
		}

		// Message flush!
		err = client.writer.Flush()
		if err != nil {
			fmt.Println("server: flush error:", err)
			return
		}
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

	// Accept a single connection
	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("server: accept error:", err)
		return
	}

	// Create a client from the connection
	client := &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
	defer client.conn.Close()

	fmt.Println("server: client connected from,", client.conn.RemoteAddr())

	// Handle the client
	handleClient(client)
}
