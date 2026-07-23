// server
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"lancom/protocol"
	"net"
	"os"
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
// dmHandler: resolves comma-separated recipient nicknames in msg.To, forwards the
// message to each one found, and reports delivery status back to the sender
func dmHandler(msg *protocol.Message, sender *Client) error {
	recipientNicks := strings.Split(msg.To, ",")

	mutexLock.Lock()
	recipients := make([]*Client, 0, len(recipientNicks))
	var notFound []string
	for _, nick := range recipientNicks {
		nick = strings.TrimSpace(nick)
		if client, exists := nickNames[nick]; exists {
			recipients = append(recipients, client)
		} else {
			notFound = append(notFound, nick)
		}
	}
	mutexLock.Unlock()

	for _, recipient := range recipients {
		if err := messageWriter(msg, recipient); err != nil {
			// TODO: log info of the recipient that didn't receive
			continue
		}
	}

	ack := protocol.Message{
		Type: protocol.TypeDMAck,
		From: protocol.Server,
		To:   sender.id,
		Body: fmt.Sprintf("delivered to %d recipient(s)", len(recipients)),
	}
	if len(notFound) > 0 {
		ack.Body += "; not found: " + strings.Join(notFound, ", ")
	}
	return messageWriter(&ack, sender)
}

// nickHandler: validates and applies a nickname change, then announces it to the room
func nickHandler(msg *protocol.Message, client *Client) error {
	newNick := strings.TrimSpace(msg.Body)
	if len(newNick) < 3 {
		return fmt.Errorf("nickname must be at least 3 characters")
	}

	mutexLock.Lock()
	if _, taken := nickNames[newNick]; taken {
		mutexLock.Unlock()
		return fmt.Errorf("nickname %q is already taken", newNick)
	}
	if _, reserved := reservedNames[strings.ToLower(newNick)]; reserved {
		mutexLock.Unlock()
		return fmt.Errorf("nickname %q is reserved", newNick)
	}

	oldNick := client.nickName
	delete(nickNames, oldNick)
	client.nickName = newNick
	nickNames[newNick] = client
	mutexLock.Unlock()

	ack := protocol.Message{
		Type: protocol.TypeNickAck,
		From: protocol.Server,
		To:   client.id,
		Body: newNick,
	}
	if err := messageWriter(&ack, client); err != nil {
		return err
	}

	broadcastMessage(&protocol.Message{
		Type: protocol.TypeChat,
		From: protocol.Server,
		To:   protocol.All,
		Body: fmt.Sprintf("%s is now known as %s", oldNick, newNick),
	}, nil)
	return nil
}

// listHandler: reports the nicknames of all currently connected clients
func listHandler(client *Client) error {
	mutexLock.Lock()
	online := make([]string, 0, len(nickNames))
	for nick := range nickNames {
		online = append(online, nick)
	}
	mutexLock.Unlock()

	ack := protocol.Message{
		Type: protocol.TypeListAck,
		From: protocol.Server,
		To:   client.id,
		Body: strings.Join(online, ","),
	}
	return messageWriter(&ack, client)
}

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
		nickName := client.nickName
		delete(nickNames, nickName)
		client.conn.Close()
		delete(clients, client)
		remaining := len(clients)
		mutexLock.Unlock()

		fmt.Printf("[clients: %d]\n", remaining)
		fmt.Printf("%s left\n", nickName)
		broadcastMessage(&protocol.Message{
			Type: protocol.TypeChat,
			From: protocol.Server,
			To:   protocol.All,
			Body: fmt.Sprintf("%s left the room", nickName),
		}, nil)
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
		return chatHandler(msgObj, client)
	case protocol.TypeDM:
		return dmHandler(msgObj, client)
	case protocol.TypeNickReq:
		return nickHandler(msgObj, client)
	case protocol.TypeListReq:
		return listHandler(client)
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
func initialSetup(addr string) error {
	var err error
	listener, err = net.Listen("tcp", addr)
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
	addr := flag.String("addr", ":9000", "address to listen on (host:port); host empty means all interfaces")
	flag.Parse()

	// initial setup
	err := initialSetup(*addr)
	if err != nil {
		fmt.Println("initial setup error, ", err)
		os.Exit(1)
	}

	fmt.Printf("listening on %s\n", *addr)

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
