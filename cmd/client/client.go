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
	fmt.Printf("%s> %s\n> ", m.From, m.Body)
}

// messageWrite: does versioning, encoding, writing of the message object
func messageWriter(msg *protocol.Message) error {
	msg.Version = protocol.Version

	data, err := protocol.Encode(msg)
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

// messageHandler: decides on what to do of the received message based on type of message
func messageHandler(msg *string) error {
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
		fmt.Println("invalid message type")
	}

	return nil
}

// connectionHandler: does the infinite i/o to the server
func connectionHandler() error {
	// persistant read/write tunnel
	for {
		msg, err := connReader.ReadString('\n')
		if err != nil {
			return err
		}
		err = messageHandler(&msg)
		if err != nil {
			return err
		}

		// TODO: implement gracefull exit from loop
	}
}

func joinHandler() error {
	// join request object
	joinObj := protocol.Message{
		Type: protocol.TypeJoinReq,
		From: "client",
		To:   "server",
	}

	// request to join
	err := messageWriter(&joinObj)
	if err != nil {
		return err
	}

	msg, err := connReader.ReadString('\n')
	if err != nil {
		return err
	}

	err = messageHandler(&msg)
	if err != nil {
		return err
	}

	return nil
}

// initialSetup: does all the initial setup, like searver search, resource allocation etc...
func initialSetup() error {
	// server search
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		return err
	}

	// resource allocation
	connReader = bufio.NewReader(conn)
	connWriter = bufio.NewWriter(conn)
	stdinReader = bufio.NewReader(os.Stdin)

	// request to join
	err = joinHandler()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// initial setup
	err := initialSetup()
	if err != nil {
		fmt.Println("initial setup error, ", err)
		// TODO: do the gracefull shutdown if error in initial setup
	}

	// TODO: you need to close `conn` after the exit
	// TODO: customize the errors for each types

	go connectionHandler()

	// persistant chat input
	for {
		fmt.Print("> ")
		msg, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("stdin read error", err)
			return
		}
		msg = strings.TrimSuffix(msg, "\n")

		msgObj := protocol.Message{
			Type: protocol.TypeChat,
			From: myId,
			To:   "server",
			Body: msg,
		}

		err = messageWriter(&msgObj)
		if err != nil {
			fmt.Println("error writing error", err)
			return
		}
	}
}
