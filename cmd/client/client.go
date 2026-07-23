// Client
package main

import (
	"bufio"
	"flag"
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

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(0, 1)

	messagesStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			AlignVertical(lipgloss.Bottom)

	selfColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green: your own messages
	peerColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // white: broadcasts from others
	dmColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("212")) // pink: private messages
	systemColor = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // gray: acks/roster/system
	errorColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // red: errors
)

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
			sendLeave()
			return m, tea.Quit
		case "enter":
			trimmed := strings.TrimSpace(m.input)
			if trimmed == "/quit" {
				sendLeave()
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
			// Only append actual typed characters (letters/digits/symbols and
			// space bar). Without this guard, any non-printable key (arrows,
			// ctrl+u, tab, ...) falls through to String() and inserts its
			// literal name (e.g. "ctrl+u") as text; space bar is its own
			// KeyType distinct from KeyRunes, so it needs listing explicitly.
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.input += msg.String()
			}
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

	// Calculate available space: 1 row for the header, 3 for the bordered
	// input box (2 border + 1 content), whatever's left for the message box.
	headerHeight := 1
	inputHeight := 3
	messagesBoxHeight := max(m.height-headerHeight-inputHeight, 3)
	contentHeight := messagesBoxHeight - 2 // minus the message box's own border

	// Keep only as many messages as fit; AlignVertical(Bottom) then keeps
	// them anchored just above the input box instead of floating at the
	// top of a mostly-empty pane.
	messagesToShow := m.messages
	if len(messagesToShow) > contentHeight {
		messagesToShow = messagesToShow[len(messagesToShow)-contentHeight:]
	}

	header := headerStyle.Render(fmt.Sprintf("lancom · %s", myId))

	messagesView := messagesStyle.
		Width(m.width - 4).
		Height(contentHeight).
		Render(strings.Join(messagesToShow, "\n"))

	inputView := inputStyle.
		Width(m.width - 4).
		Render(fmt.Sprintf("➜ %s", m.input))

	return fmt.Sprintf("%s\n%s\n%s", header, messagesView, inputView)
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
		return append(messages, errorColor.Render(fmt.Sprintf("Error: %v", err)))
	}
	return append(messages, selfColor.Render(fmt.Sprintf("you> %s", input)))
}

// runCommand: parses a "/"-prefixed input line into the matching protocol request
func runCommand(input string, messages []string) []string {
	cmd, rest, _ := strings.Cut(input, " ")

	switch cmd {
	case "/msg":
		recipients, body, ok := strings.Cut(rest, " ")
		if !ok || recipients == "" || body == "" {
			return append(messages, systemColor.Render("usage: /msg <recipient[,recipient...]> <message>"))
		}
		msg := &protocol.Message{
			Type: protocol.TypeDM,
			From: myId,
			To:   recipients,
			Body: body,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, errorColor.Render(fmt.Sprintf("Error: %v", err)))
		}
		return append(messages, selfColor.Render(fmt.Sprintf("you -> %s> %s", recipients, body)))

	case "/nick":
		if rest == "" {
			return append(messages, systemColor.Render("usage: /nick <new-name>"))
		}
		msg := &protocol.Message{
			Type: protocol.TypeNickReq,
			From: myId,
			To:   protocol.Server,
			Body: rest,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, errorColor.Render(fmt.Sprintf("Error: %v", err)))
		}
		return messages

	case "/list":
		msg := &protocol.Message{
			Type: protocol.TypeListReq,
			From: myId,
			To:   protocol.Server,
		}
		if err := messageWriter(msg); err != nil {
			return append(messages, errorColor.Render(fmt.Sprintf("Error: %v", err)))
		}
		return messages

	default:
		return append(messages, systemColor.Render(fmt.Sprintf("unknown command: %s", cmd)))
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
			case protocol.TypeChatAck, protocol.TypeNickAck:
				// chat_ack is a silent sender-only receipt; nick_ack is made
				// redundant by the room-wide "X is now known as Y" broadcast
				// the renamer also receives, so neither needs its own line.
				continue
			case protocol.TypeChat:
				return msgReceived(peerColor.Render(fmt.Sprintf("<%s> %s", msgObj.From, msgObj.Body)))
			case protocol.TypeDM:
				return msgReceived(dmColor.Render(fmt.Sprintf("[DM from %s] %s", msgObj.From, msgObj.Body)))
			case protocol.TypeDMAck:
				return msgReceived(systemColor.Render(fmt.Sprintf("* %s", msgObj.Body)))
			case protocol.TypeListAck:
				return msgReceived(systemColor.Render(fmt.Sprintf("* online: %s", strings.ReplaceAll(msgObj.Body, ",", ", "))))
			case protocol.ErrorMessage:
				return msgReceived(errorColor.Render(fmt.Sprintf("* error: %s", msgObj.Body)))
			default:
				return msgReceived(systemColor.Render(fmt.Sprintf("* unhandled message type: %s", msgObj.Type)))
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

// sendLeave: tells the server we're leaving instead of just dropping the
// connection, so other clients get an immediate "left the room" notice
func sendLeave() {
	messageWriter(&protocol.Message{
		Type: protocol.TypeLeave,
		From: myId,
		To:   protocol.Server,
	})
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
	server := flag.String("server", "127.0.0.1:9000", "address of the lancom server (host:port)")
	flag.Parse()

	conn, err := net.Dial("tcp", *server)
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
