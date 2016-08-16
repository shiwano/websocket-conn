package conn

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

var (
	// ErrMessageChannelFull is returned when the connection's envelope channel is full.
	ErrMessageChannelFull = errors.New("Message channel is full")

	closeEnvelope          = &envelope{websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")}
	closeGoingAwayEnvelope = &envelope{websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "")}
	pingEnvelope           = &envelope{websocket.PingMessage, []byte{}}
	pongEnvelope           = &envelope{websocket.PongMessage, []byte{}}
)

// Conn represents a web socket connection.
type Conn struct {
	Settings             *Settings
	BinaryMessageHandler func([]byte)
	TextMessageHandler   func(string)
	DisconnectHandler    func()
	ErrorHandler         func(error)

	conn       *websocket.Conn
	envelopeCh chan *envelope
	mu         *sync.Mutex
}

// New creates a Conn.
func New() *Conn {
	return &Conn{
		Settings: newDefaultSettings(),
		mu:       new(sync.Mutex),
	}
}

// Connect to the peer.
func (c *Conn) Connect(url string, requestHeader http.Header) (*http.Response, error) {
	c.envelopeCh = make(chan *envelope, c.Settings.MessageChannelBufferSize)

	dialer := new(websocket.Dialer)
	dialer.ReadBufferSize = c.Settings.ReadBufferSize
	dialer.WriteBufferSize = c.Settings.WriteBufferSize
	dialer.HandshakeTimeout = c.Settings.HandshakeTimeout
	dialer.Subprotocols = c.Settings.Subprotocols
	dialer.NetDial = c.Settings.DialerSettings.NetDial
	dialer.TLSClientConfig = c.Settings.DialerSettings.TLSClientConfig

	conn, response, err := dialer.Dial(url, requestHeader)
	if err != nil {
		return response, err
	}
	c.conn = conn

	go c.writePump()
	go c.readPump()
	return response, nil
}

// UpgradeFromHTTP upgrades HTTP to WebSocket.
func (c *Conn) UpgradeFromHTTP(responseWriter http.ResponseWriter, request *http.Request) error {
	c.envelopeCh = make(chan *envelope, c.Settings.MessageChannelBufferSize)

	upgrader := new(websocket.Upgrader)
	upgrader.ReadBufferSize = c.Settings.ReadBufferSize
	upgrader.WriteBufferSize = c.Settings.WriteBufferSize
	upgrader.HandshakeTimeout = c.Settings.HandshakeTimeout
	upgrader.Subprotocols = c.Settings.Subprotocols
	upgrader.Error = c.Settings.UpgraderSettings.Error
	upgrader.CheckOrigin = c.Settings.UpgraderSettings.CheckOrigin

	conn, err := upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		return err
	}
	c.conn = conn

	go c.writePump()
	go c.readPump()
	return nil
}

// Close the connection.
func (c *Conn) Close() error {
	return c.postEnvelope(closeEnvelope)
}

// WriteBinaryMessage to the peer.
func (c *Conn) WriteBinaryMessage(data []byte) error {
	return c.postEnvelope(&envelope{websocket.BinaryMessage, data})
}

// WriteTextMessage to the peer.
func (c *Conn) WriteTextMessage(text string) error {
	return c.postEnvelope(&envelope{websocket.TextMessage, []byte(text)})
}

func (c *Conn) postEnvelope(e *envelope) error {
	select {
	case c.envelopeCh <- e:
		return nil
	default:
		return ErrMessageChannelFull
	}
}

func (c *Conn) writeMessage(e *envelope) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.Settings.WriteWait)); err != nil {
		c.handleError(err)
		return err
	}
	if err := c.conn.WriteMessage(e.messageType, e.data); err != nil {
		c.handleError(err)
		return err
	}
	return nil
}

func (c *Conn) handleError(err error) {
	if err != nil && c.ErrorHandler != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.ErrorHandler(err)
	}
}

func (c *Conn) writePump() {
	defer c.conn.Close()

	ticker := time.NewTicker(c.Settings.PingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case e, ok := <-c.envelopeCh:
			if !ok {
				c.writeMessage(closeGoingAwayEnvelope)
				break loop
			}
			if err := c.writeMessage(e); err != nil {
				break loop
			}
		case <-ticker.C:
			if err := c.writeMessage(pingEnvelope); err != nil {
				break loop
			}
		}
	}
}

func (c *Conn) readPump() {
	defer c.conn.Close()

	c.conn.SetReadLimit(c.Settings.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.Settings.PongWait))
	c.conn.SetPingHandler(func(string) error {
		return c.postEnvelope(pongEnvelope)
	})
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.Settings.PongWait))
	})

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			c.handleError(err)
			break
		}

		switch messageType {
		case websocket.BinaryMessage:
			if c.BinaryMessageHandler != nil {
				c.BinaryMessageHandler(data)
			}
		case websocket.TextMessage:
			if c.TextMessageHandler != nil {
				c.TextMessageHandler(string(data))
			}
		}
	}

	if c.DisconnectHandler != nil {
		c.DisconnectHandler()
	}
}
