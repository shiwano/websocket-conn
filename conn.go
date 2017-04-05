package conn

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrMessageChannelFull indicates that the connection's envelope channel is full.
	ErrMessageChannelFull = errors.New("websocket-conn: Message channel is full")

	// ErrAlreadyUsed indicates that the connection is already used.
	ErrAlreadyUsed = errors.New("websocket-conn: Already used")

	closeEnvelope = envelope{websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")}
	pingEnvelope  = envelope{websocket.PingMessage, []byte{}}
	pongEnvelope  = envelope{websocket.PongMessage, []byte{}}
)

// Conn represents a WebSocket connection.
type Conn struct {
	Settings             *Settings
	BinaryMessageHandler func([]byte)
	TextMessageHandler   func(string)
	DisconnectionHandler func(error)

	conn              *websocket.Conn
	envelopeCh        chan envelope
	writePumpFinishCh chan error
	readPumpFinishCh  chan struct{}
	ctx               context.Context
}

// New creates a Conn.
func New(ctx context.Context) *Conn {
	return &Conn{
		Settings: newDefaultSettings(),
		ctx:      ctx,
	}
}

// Connect to the peer.
func (c *Conn) Connect(url string, requestHeader http.Header) (*http.Response, error) {
	if c.conn != nil {
		return nil, ErrAlreadyUsed
	}
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
	c.start()
	return response, nil
}

// UpgradeFromHTTP upgrades HTTP to WebSocket.
func (c *Conn) UpgradeFromHTTP(responseWriter http.ResponseWriter, request *http.Request) error {
	if c.conn != nil {
		return ErrAlreadyUsed
	}
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
	c.start()
	return nil
}

// WriteBinaryMessage to the peer.
func (c *Conn) WriteBinaryMessage(data []byte) error {
	return c.postEnvelope(envelope{websocket.BinaryMessage, data})
}

// WriteTextMessage to the peer.
func (c *Conn) WriteTextMessage(text string) error {
	return c.postEnvelope(envelope{websocket.TextMessage, []byte(text)})
}

func (c *Conn) start() {
	c.writePumpFinishCh = make(chan error, 1)
	c.readPumpFinishCh = make(chan struct{}, 1)
	c.envelopeCh = make(chan envelope, c.Settings.MessageChannelBufferSize)

	c.conn.SetReadLimit(c.Settings.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.Settings.PongWait))
	c.conn.SetPingHandler(func(string) error {
		return c.postEnvelope(pongEnvelope)
	})
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.Settings.PongWait))
	})

	go c.writePump()
	go c.readPump()
}

func (c *Conn) postEnvelope(e envelope) error {
	select {
	case c.envelopeCh <- e:
		return nil
	default:
		return ErrMessageChannelFull
	}
}

func (c *Conn) writeMessage(e envelope) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.Settings.WriteWait)); err != nil {
		return err
	}
	if err := c.conn.WriteMessage(e.messageType, e.data); err != nil {
		return err
	}
	return nil
}

func (c *Conn) writePump() {
	defer c.conn.Close()

	ticker := time.NewTicker(c.Settings.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			err := c.writeMessage(closeEnvelope)
			if err == nil {
				err = c.ctx.Err()
			}
			c.writePumpFinishCh <- err
			return
		case <-c.readPumpFinishCh:
			return
		case e := <-c.envelopeCh:
			if err := c.writeMessage(e); err != nil {
				c.writePumpFinishCh <- err
				return
			}
		case <-ticker.C:
			if err := c.writeMessage(pingEnvelope); err != nil {
				c.writePumpFinishCh <- err
				return
			}
		}
	}
}

func (c *Conn) readPump() {
	defer c.conn.Close()
	var disconnectionError error

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			disconnectionError = err
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

	select {
	case disconnectionError = <-c.writePumpFinishCh:
	default:
		c.readPumpFinishCh <- struct{}{}
	}

	if c.DisconnectionHandler != nil {
		c.DisconnectionHandler(disconnectionError)
	}
	close(c.envelopeCh)
}
