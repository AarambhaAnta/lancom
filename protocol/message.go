package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	TypeJoin    = "join"
	TypeJoinAck = "join_ack"
	TypeChat    = "chat"
	TypeChatAck = "chat_ack"
	TypeLeave   = "leave"
)

// Protocol knows JSON
// Transport knows `\n`
type Message struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
	Body string `json:"body"`
}

// JSON to Message
func Decode(data []byte) (*Message, error) {
	msg := Message{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Message to JSON
func Encode(m *Message) ([]byte, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return data, err
}

// Validation after decode []byte -> Message{}
func (m *Message) Validate() error {
	switch m.Type {
	case TypeJoin:
		if m.Body != "" {
			return fmt.Errorf("invalid message body for type %s", TypeJoin)
		}
	case TypeJoinAck:
		if m.Body == "" {
			return fmt.Errorf("invalid message body for type acknowledgement %s", TypeJoinAck)
		}
	case TypeChat:
		if m.Body == "" {
			return fmt.Errorf("invalid message body for type %s", TypeChat)
		}
	case TypeChatAck:
		if m.Body == "" {
			return fmt.Errorf("invalid message body for chat acknowledgement %s", TypeChatAck)
		}
	case TypeLeave:
		return nil
	default:
		return errors.New("invalid message type")
	}
	return nil
}
