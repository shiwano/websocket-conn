package conn

import (
	"github.com/gorilla/websocket"
)

// MessageType represents websocket message types.
type MessageType int

// MessageType constants. these are same websocket package's values.
const (
	TextMessageType   = websocket.TextMessage
	BinaryMessageType = websocket.BinaryMessage
)

// Message represents a message.
type Message struct {
	MessageType MessageType
	Data        []byte
}

// Text returns text message data as string.
func (m Message) Text() string {
	if m.MessageType == TextMessageType {
		return string(m.Data)
	}
	return ""
}
