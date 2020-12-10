package wsconn

import (
	"encoding/json"

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

// IsTextMessage determines whether the message is a text message.
func (m Message) IsTextMessage() bool {
	return m.MessageType == TextMessageType
}

// IsBinaryMessage determines whether the message is a binary message.
func (m Message) IsBinaryMessage() bool {
	return m.MessageType == BinaryMessageType
}

// Text returns text message data as string.
func (m Message) Text() string {
	if m.MessageType == TextMessageType {
		return string(m.Data)
	}
	return ""
}

// UnmarshalAsJSON unmarshals data as JSON.
func (m Message) UnmarshalAsJSON(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}
