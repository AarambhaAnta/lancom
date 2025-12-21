// Client
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("client: failed to connect")
		panic(err)
	}
	defer conn.Close()

	fmt.Println("client: connected to server")

	// Create readers and writers
	connReader := bufio.NewReader(conn)
	connWriter := bufio.NewWriter(conn)
	stdinReader := bufio.NewReader(os.Stdin)

	// Start a goroutine to read messages from the server
	go func() {
		for {
			response, err := connReader.ReadString('\n')
			if err != nil {
				fmt.Println("\nclient: server disconnected")
				os.Exit(0)
				return
			}
			fmt.Print("\r↓ ", response, "> ")
		}
	}()

	// Main loop to read from stdin and send to server
	for {
		fmt.Print("> ")
		message, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("client: read error:", err)
			return
		}

		// Send to server
		_, err = connWriter.WriteString(message)
		if err != nil {
			fmt.Println("client: write error:", err)
			return
		}
		connWriter.Flush()
	}
}
