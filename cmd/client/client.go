// Client
package main

import (
	"bufio"
	"fmt"
	"lancom/protocol"
	"net"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	connReader *bufio.Reader
	connWriter *bufio.Writer
	myId       string
)

type model struct {
	messages []string
	input    string
	width    int
	height   int
	conn     net.Conn
	ready    bool
}

type msgReceived string

var inputStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(0, 1)

var messagesStyle = lipgloss.NewStyle().
	Padding(1, 2)

func initialModel(conn net.Conn) model {
	return model{
		messages: []string{},
		input:    "",
		conn:     conn,
		ready:    false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForMessages(m.conn),
		tea.EnterAltScreen,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			trimmed := strings.TrimSpace(m.input)
			if trimmed == "/quit" {
				return m, tea.Quit
			}
			if trimmed != "" {
				m.messages = handleSubmit(trimmed, m.messages)
			}
			m.input = ""
			return m, nil
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
			return m, nil
		default:
			m.input += msg.String()
			return m, nil
		}

	case msgReceived:
		m.messages = append(m.messages, string(msg))
		return m, waitForMessages(m.conn)

	case error:
		m.messages = append(m.messages, fmt.Sprintf("Error: %v", msg))
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate available space
	inputHeight := 3 // border + padding
	messagesHeight := m.height - inputHeight

	// Render messages (scrollable area)
	var messagesToShow []string
	startIdx := 0
	if len(m.messages) > messagesHeight-2 {
		startIdx = len(m.messages) - (messagesHeight - 2)
	}
	messagesToShow = m.messages[startIdx:]

	messagesView := messagesStyle.
		Width(m.width - 4).
		Height(messagesHeight - 2).
		Render(strings.Join(messagesToShow, "\n"))

	// Render input (fixed at bottom)
	inputView := inputStyle.
		Width(m.width - 4).
		Render(fmt.Sprintf("Message: %s", m.input))

	return fmt.Sprintf("%s\n%s", messagesView, inputView)
}

// handleSubmit: routes a submitted input line to a command or a broadcast chat
// message, sends it, and appends a local echo so the sender sees their own message
func handleSubmit(input string, messages []string) []string {
	if strings.HasPrefix(input, "/") {
		return runCommand(input, messages)
	}

	msg := &protocol.Message{
		Type: protocol.TypeChat,
		From: myId,
		To:   protocol.All,
		Body: input,
	}
	if err := messageWriter(msg); err != nil {
		return append(messages, fmt.Sprintf("Error: %v", err))
	}
	return append(messages, fmt.Sprintf("you> %s", input))
}

// runCommand: parses a "/"-prefixed input line into the matching protocol request
func runCommand(input string, messages []string) []string {
	cmd, rest, _ := strings.Cut(input, " ")

	switch cmd {
	case "/msg":
		recipients, body, ok := strings.Cut(rest, " ")
		if !ok || recipients == "" || body == "" {
			return append(messages, "usage: /msg <recipient[,recipient...]> <message>")
		}
		msg := &protocol.Message{
			Type: protocol.TypeDM,
			From: myId,
			To:   recipients,
			Body: body,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, fmt.Sprintf("Error: %v", err))
		}
		return append(messages, fmt.Sprintf("you -> %s> %s", recipients, body))

	case "/nick":
		if rest == "" {
			return append(messages, "usage: /nick <new-name>")
		}
		msg := &protocol.Message{
			Type: protocol.TypeNickReq,
			From: myId,
			To:   protocol.Server,
			Body: rest,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, fmt.Sprintf("Error: %v", err))
		}
		return messages

	case "/list":
		msg := &protocol.Message{
			Type: protocol.TypeListReq,
			From: myId,
			To:   protocol.Server,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, fmt.Sprintf("Error: %v", err))
		}
		return messages

	default:
		return append(messages, fmt.Sprintf("unknown command: %s", cmd))
	}
}

// waitForMessages: blocks for the next line from the server and turns it into a
// tea.Msg, formatted according to its protocol message type
func waitForMessages(conn net.Conn) tea.Cmd {
	return func() tea.Msg {
		// Loops past ack-only message types instead of returning nil: a nil
		// tea.Msg would end the command chain and the client would stop
		// reading from the socket after the very first chat_ack.
		for {
			line, err := connReader.ReadString('\n')
			if err != nil {
				return error(fmt.Errorf("server disconnected: %w", err))
			}

			msgObj, err := protocol.Decode([]byte(line))
			if err != nil {
				return error(err)
			}
			if err := msgObj.Validate(); err != nil {
				return error(err)
			}

			switch msgObj.Type {
			case protocol.TypeChatAck:
				continue
			case protocol.TypeChat:
				return msgReceived(fmt.Sprintf("<%s> %s", msgObj.From, msgObj.Body))
			case protocol.TypeDM:
				return msgReceived(fmt.Sprintf("[DM from %s] %s", msgObj.From, msgObj.Body))
			case protocol.TypeDMAck:
				return msgReceived(fmt.Sprintf("* %s", msgObj.Body))
			case protocol.TypeNickAck:
				return msgReceived(fmt.Sprintf("* you are now known as %s", msgObj.Body))
			case protocol.TypeListAck:
				return msgReceived(fmt.Sprintf("* online: %s", strings.ReplaceAll(msgObj.Body, ",", ", ")))
			case protocol.ErrorMessage:
				return msgReceived(fmt.Sprintf("* error: %s", msgObj.Body))
			default:
				return msgReceived(fmt.Sprintf("* unhandled message type: %s", msgObj.Type))
			}
		}
	}
}

// messageWriter: does versioning, encoding, and writing of the message object
func messageWriter(m *protocol.Message) error {
	m.Version = protocol.Version
	data, err := protocol.Encode(m)
	if err != nil {
		return err
	}
	if _, err := connWriter.WriteString(string(data) + "\n"); err != nil {
		return err
	}
	return connWriter.Flush()
}

// joinHandler: performs the initial join handshake with the server
func joinHandler() error {
	joinReq := &protocol.Message{
		Type: protocol.TypeJoinReq,
		From: "client",
		To:   protocol.Server,
	}
	if err := messageWriter(joinReq); err != nil {
		return fmt.Errorf("failed to request join: %w", err)
	}

	line, err := connReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read join ack: %w", err)
	}

	msgObj, err := protocol.Decode([]byte(line))
	if err != nil {
		return err
	}
	if msgObj.Type == protocol.TypeJoinAck {
		myId = msgObj.Body
	}
	return nil
}

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("client: failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	connReader = bufio.NewReader(conn)
	connWriter = bufio.NewWriter(conn)

	if err := joinHandler(); err != nil {
		fmt.Println("client: handshake failed:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(conn), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
