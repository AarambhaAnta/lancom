package protocol

import (
	"encoding/json"
	"errors"
)

const (
	TypeJoinReq  = "join_req"
	TypeJoinAck  = "join_ack"
	TypeChat     = "chat"
	TypeChatAck  = "chat_ack"
	TypeDM       = "dm"
	TypeDMAck    = "dm_ack"
	TypeNickReq  = "nick_req"
	TypeNickAck  = "nick_ack"
	TypeListReq  = "list_req"
	TypeListAck  = "list_ack"
	TypeLeave    = "leave"
	ErrorMessage = "error_message"
	Server       = "server"
	All          = "all"
	Version      = "1.0"
)

// Protocol knows JSON
// Transport knows `\n`
type Message struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	From    string `json:"from"`
	To      string `json:"to"`
	Body    string `json:"body"`
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
	if m.Version != Version {
		return errors.New("unsupported version")
	}
	if m.Type == "" {
		return errors.New("message type cannot be empty")
	}
	if m.Type == TypeDM && m.To == "" {
		return errors.New("dm requires at least one recipient")
	}
	return nil
}
