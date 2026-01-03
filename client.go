// Client
package main

import (
	"bufio"
	"fmt"
	"lancom/protocol"
	"net"
	"os"
)

func show(m *protocol.Message) {
	fmt.Printf("<⬇ %s> %s\n", m.From, m.Body)
}

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

	// Start a goroutine to read string(msgJson)s from the server
	go func() {
		for {
			response, err := connReader.ReadString('\n')
			if err != nil {
				fmt.Println("\nclient: server disconnected")
				os.Exit(0)
				return
			}

			// Message object
			message, err := protocol.Decode([]byte(response))
			if err != nil {
				fmt.Println("client: message decode error: ", err)
			}
			err = message.Validate()
			if err != nil {
				fmt.Println("client: Message validation error: ", err)
			}

			show(message)
		}
	}()

	// Main loop to read from stdin and send to server
	for {
		fmt.Printf("<⬆ %s> ", "server")
		msg, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("client: read error:", err)
			return
		}

		// Build Message object -> json -> send to server
		message := protocol.Message{
			Type: "chat",
			From: "check",
			To:   "server",
			Body: msg,
		}

		msgByte, err := protocol.Encode(&message)
		if err != nil {
			fmt.Println("client: parse error", err)
			continue
		}

		_, err = connWriter.WriteString(string(msgByte) + "\n")
		if err != nil {
			fmt.Println("client: write error:", err)
			return
		}
		connWriter.Flush()
	}
}
