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

var (
	clients         map[*Client]bool
	nickNames       map[string]*Client
	mutexLock       sync.Mutex
	clientIDCounter uint64
	reservedNames   map[string]struct{}
	listener        net.Listener
)

// Client: is a struct with all the related data for a client
type Client struct {
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	id       string
	isJoined bool
	nickName string
}

// getNextClientID: generates a sequential, unique client ID with atomic operations meaning thread-saftey and no dublicates
func getNextClientID() string {
	id := atomic.AddUint64(&clientIDCounter, 1)
	return fmt.Sprintf("client-%d", id)
}

// messageWriter: does versioning, encoding, and writing to the connection
func messageWriter(msg *protocol.Message, client *Client) error {
	msg.Version = protocol.Version

	data, err := protocol.Encode(msg)
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

// broadcastMessage: broadcast message to all-except-sender or to a specific sender
// TODO: you need to implement the worker in go routine for writing message simultenously
func broadcastMessage(msg *protocol.Message, sender *Client) {
	// make list of currently connected clients
	mutexLock.Lock()
	clientList := make([]*Client, 0, len(clients))
	for client := range clients {
		if client != sender {
			clientList = append(clientList, client)
		}
	}
	mutexLock.Unlock()

	// send message to clients that were connected at that moment
	for _, client := range clientList {
		err := messageWriter(msg, client)
		if err != nil {
			// TODO: log info of the client that didn't receive
			continue
		}
	}
}
// // singleMessage is a personal message to a particular client
// func singleMessage(msg *protocol.Message) error {
// 	msg.Body = "@abhi hi,ola"
// 	parts := strings.SplitN(msg.Body, " ", 2)
// 		if len(parts) < 2 {
// 			return fmt.Errorf("invalid message format")
// 		}
// 		recipientNick := strings.TrimPrefix(parts[0], "@")
// 		messageBody := parts[1]
		
// 		mutexLock.Lock()
// 		recipient, exists := nickNames[recipientNick]
// 		mutexLock.Unlock()
		
// 		if !exists {
// 			return fmt.Errorf("user %s not found", recipientNick)
// 		}
		
// 		msg.To = recipient.id
// 		msg.Body = messageBody
// 	return messageWriter(msg, nickNames[msg.To])
// }

// // Nick Command: command handler that handles the change of nick names for a client
// func nickCommand(args []string, client *Client) error {
// 	newNick := args[0]
// 	mu.Lock()
// 	oldNick := client.nick
// 	client.nick = newNick
// 	nicks[newNick] = client
// 	delete(nicks, oldNick)
// 	mu.Unlock()

// 	// broadcast change of nick name
// 	broadcaster(&protocol.Message{
// 		Type: protocol.TypeChat,
// 		From: protocol.Server,
// 		To:   protocol.All,
// 		Body: fmt.Sprintf("%s -> %s", oldNick, newNick),
// 	}, nil)

// 	return nil
// }

// // Command Parser: used to parse a string into command and arguments
// func commandParser(cmd string) (string, []string, error) {
// 	parts := strings.Fields(cmd)
// 	if len(parts) == 0 {
// 		return "", nil, fmt.Errorf("not a valid command")
// 	}
// 	return parts[0], parts[1:], nil
// }

// // Command Validate: validates all the commands before executing
// func commandValidate(cmd string, args []string) error {
// 	switch cmd {
// 	case "/nick":
// 		if len(args) == 0 {
// 			return fmt.Errorf("invalid arguments")
// 		}
// 		nick := args[0]

// 		if len(nick) < 3 {
// 			return fmt.Errorf("nick name too short")
// 		}
// 		if _, exists := nicks[nick]; exists {
// 			return fmt.Errorf("nick name already taken")
// 		}
// 		if _, ok := reservedName[strings.ToLower(nick)]; ok {
// 			return fmt.Errorf("nick name is reserved")
// 		}
// 		return nil
// 	default:
// 		return fmt.Errorf("unknown command")
// 	}
// }

// // Command handler: handles different types of command like `/nick`, `/who`
// func commandHandler(msg *protocol.Message, client *Client) error {
// 	cmd, args, err := commandParser(msg.Body)
// 	if err != nil {
// 		return err
// 	}

// 	err = commandValidate(cmd, args)
// 	if err != nil {
// 		return err
// 	}

// 	switch cmd {
// 	case "/nick":
// 		return nickCommand(args, client)
// 	default:
// 		return messageWriter(&protocol.Message{
// 			Type: protocol.TypeChatAck,
// 			From: protocol.Server,
// 			To:   client.id,
// 			Body: "unknown command",
// 		}, client)
// 	}
// }

// Chat handler: handles what do on chat request
func chatHandler(msg *protocol.Message, client *Client) error {
	broadcastMessage(msg, client)
	msgAck := protocol.Message{
		Type: protocol.TypeChatAck,
		From: protocol.Server,
		To:   client.id,
		Body: "Message sent to all...",
	}

	err := messageWriter(&msgAck, client)
	if err != nil {
		return err
	}

	return nil
}

// leaveHandler: used to release resouces allocated to a particular client
func leaveHandler(client *Client) error {
	// release all the resources allocated to a client
	mutexLock.Lock()
	if _, exists := clients[client]; exists && client.isJoined {
		delete(nickNames, client.nickName)
		client.conn.Close()
		delete(clients, client)
		mutexLock.Unlock()
		return nil
	}
	mutexLock.Unlock()

	return nil
}

// joinHandler: used to do the initial setup for a client and sent a join acknowledgment
func joinHandler(client *Client) error {
	// establish all the initial setup for a client
	mutexLock.Lock()
	if client.isJoined {
		return fmt.Errorf("client has already joined")
	}
	clients[client] = true
	client.id = getNextClientID()
	client.isJoined = true
	client.nickName = client.id
	nickNames[client.nickName] = client
	mutexLock.Unlock()

	fmt.Printf("[clients: %d]\n", len(clients))
	fmt.Printf("%s joined\n", client.nickName)
	msg := protocol.Message{
		Type: protocol.TypeJoinAck,
		From: "server",
		To:   client.id,
		Body: client.id,
	}

	return messageWriter(&msg, client)
}

// semanticValidator: used to do the semantic validation from the server side
func semanticValidator(m *protocol.Message, client *Client) error {
	if m.Type == protocol.TypeJoinAck && m.From != protocol.Server {
		return fmt.Errorf("acknowledgement can only be sent by server")
	}
	if m.Type == protocol.TypeLeave && !client.isJoined {
		return fmt.Errorf("%s is not joined", client.id)
	}

	return nil
}

// messageHandler: does decoding, validation, semantic-validation and then message type based re-routing request
func messageHandler(msg *string, client *Client) error {
	msgObj, err := protocol.Decode([]byte(*msg))
	if err != nil {
		return err
	}
	err = msgObj.Validate()
	if err != nil {
		return err
	}

	if !client.isJoined && msgObj.Type != protocol.TypeJoinReq {
		return errors.New("client must join first")
	}

	err = semanticValidator(msgObj, client)
	if err != nil {
		return err
	}

	msgObj.From = client.nickName

	// route the request to specific handler
	switch msgObj.Type {
	case protocol.TypeJoinReq:
		return joinHandler(client)
	case protocol.TypeChat:
		if strings.HasPrefix(msgObj.Body, "/") {
			// return commandHandler(msgObj, client)
			return nil
		}
		if strings.HasPrefix(msgObj.Body, "@") {
			// TODO: handle single person/personal message
			return nil
		}
		return chatHandler(msgObj, client)
	case protocol.TypeLeave:
		return leaveHandler(client)
	}
	return nil
}

// clientHandler: helps in establishing persistent read tunnel and calls the relivant function for resource deallocation
func clientHandler(client *Client) {
	defer leaveHandler(client)

	// persistent read on the client
	for {
		msg, err := client.reader.ReadString('\n')
		if err != nil {
			// TODO: handle this if error occurs too-many times, exit and trigger defered cleanup
			return
		}

		msg = strings.TrimSuffix(msg, "\n")

		err = messageHandler(&msg, client)
		if err != nil {
			fmt.Printf("failed to process message from: %s, %v\n", client.id, err)
			errMsg := protocol.Message{
				Type: protocol.ErrorMessage,
				From: protocol.Server,
				To:   client.id,
				Body: string(err.Error()),
			}
			messageWriter(&errMsg, client)
		}
	}
}

func makeNewClient(conn *net.Conn) *Client {
	client := &Client{
		conn:   *conn,
		reader: bufio.NewReader(*conn),
		writer: bufio.NewWriter(*conn),
	}
	return client
}

// initialSetup: does all the initial setup, like firing up a listener and reserving/declaring some resources/constants
func initialSetup() error {
	var err error
	listener, err = net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		return err
	}

	// resource allocation
	clients = make(map[*Client]bool)
	nickNames = make(map[string]*Client)
	reservedNames = map[string]struct{}{
		"server": {}, "client": {}, "admin": {}, "root": {}, "system": {},
	}

	return nil
}

func main() {
	// initial setup
	err := initialSetup()
	if err != nil {
		fmt.Println("initial setup error, ", err)
		// TODO: you need to gracefull shutdown the server
	}

	fmt.Println("listening on 127.0.0.1:9000")

	// persistant loop for new client to join
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept error,", err)
			continue
		}

		client := makeNewClient(&conn)
		go clientHandler(client)
	}
}
