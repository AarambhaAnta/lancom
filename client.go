// Client
package main

import (
	"bufio"
	"fmt"
	"lancom/protocol"
	"net"
	"os"
	"strings"
)

var (
	connReader  *bufio.Reader
	connWriter  *bufio.Writer
	stdinReader *bufio.Reader
	myId        string
)

func show(m *protocol.Message) {
	fmt.Printf("<⬇ %s> %s\n", m.From, m.Body)
}

// msgWriter: writes message to the connection
func messageWriter(m *protocol.Message) error {
	data, err := protocol.Encode(m)
	if err != nil {
		return err
	}
	_, err = connWriter.WriteString(string(data) + "\n")
	if err != nil {
		return err
	}
	connWriter.Flush()
	return nil
}

// msgHandler: decides on what to do of the received msg
func msgHandler(msg *string) error {
	msgObj, err := protocol.Decode([]byte(*msg))
	if err != nil {
		return err
	}
	err = msgObj.Validate()
	if err != nil {
		return err
	}

	switch msgObj.Type {
	case protocol.TypeJoinAck:
		myId = msgObj.Body
		return nil
	case protocol.TypeChatAck:
		return nil
	case protocol.TypeChat:
		show(msgObj)
	default:
		fmt.Println("invalid message type...")
	}

	return nil
}

// serverHandler: handles the connection read/write to/from server
func serverHandler(conn *net.Conn) {
	// Request to join
	joinReq := protocol.Message{
		Type: protocol.TypeJoin,
		From: "client",
		To:   "server",
	}

	err := messageWriter(&joinReq)
	if err != nil {
		fmt.Println("client: failed to request join:", err)
		return
	}

	// persistant read/write tunnel
	for {
		msg, err := connReader.ReadString('\n')
		if err != nil {
			fmt.Println("\nclient: server disconnected")
			os.Exit(0)
			return
		}
		err = msgHandler(&msg)
		if err != nil {
			fmt.Println("client: message error:", err)
		}
	}
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
	connReader = bufio.NewReader(conn)
	connWriter = bufio.NewWriter(conn)
	stdinReader = bufio.NewReader(os.Stdin)

	go serverHandler(&conn)

	// Main loop to read from stdin and send to server
	for {
		msg, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("client: read error:", err)
			return
		}
		msg = strings.TrimSuffix(msg, "\n")

		// Build Message object -> json -> send to server
		message := protocol.Message{
			Type: "chat",
			From: myId,
			To:   "server",
			Body: msg,
		}

		err = messageWriter(&message)
		if err != nil {
			fmt.Println("client: write error:", err)
			return
		}
	}
}
