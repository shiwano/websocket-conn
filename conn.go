package wsconn

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrCloseSent is returned when the application writes a message to the
	// connection after sending a close message.
	ErrCloseSent = websocket.ErrCloseSent

	closeMessage = Message{websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")}
	pingMessage  = Message{websocket.PingMessage, []byte{}}
	pongMessage  = Message{websocket.PongMessage, []byte{}}
)

// Conn represents a WebSocket connection.
type Conn struct {
	conn *websocket.Conn
	err  error

	pingPeriod time.Duration
	writeWait  time.Duration

	writeMessageMu    sync.Mutex
	messageReceived   chan Message
	errored           chan error
	readPumpFinished  chan struct{}
	writePumpFinished chan struct{}
}

// Connect to the peer. the requestHeader argument may be nil.
func Connect(ctx context.Context, settings Settings, url string, requestHeader http.Header) (*Conn, *http.Response, error) {
	dialer := new(websocket.Dialer)
	dialer.ReadBufferSize = settings.ReadBufferSize
	dialer.WriteBufferSize = settings.WriteBufferSize
	dialer.HandshakeTimeout = settings.HandshakeTimeout
	dialer.Subprotocols = settings.Subprotocols
	dialer.NetDial = settings.DialerSettings.NetDial
	dialer.TLSClientConfig = settings.DialerSettings.TLSClientConfig

	conn, response, err := dialer.Dial(url, requestHeader)
	if err != nil {
		return nil, response, err
	}

	c, err := newConn(conn, settings)
	if err != nil {
		return nil, response, err
	}

	c.start(ctx)
	return c, response, nil
}

// UpgradeFromHTTP upgrades HTTP to WebSocket.
func UpgradeFromHTTP(ctx context.Context, settings Settings, w http.ResponseWriter, r *http.Request) (*Conn, error) {
	upgrader := new(websocket.Upgrader)
	upgrader.ReadBufferSize = settings.ReadBufferSize
	upgrader.WriteBufferSize = settings.WriteBufferSize
	upgrader.HandshakeTimeout = settings.HandshakeTimeout
	upgrader.Subprotocols = settings.Subprotocols
	upgrader.Error = settings.UpgraderSettings.Error
	upgrader.CheckOrigin = settings.UpgraderSettings.CheckOrigin

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	c, err := newConn(conn, settings)
	if err != nil {
		return nil, err
	}

	c.start(ctx)
	return c, nil
}

func newConn(conn *websocket.Conn, settings Settings) (*Conn, error) {
	c := &Conn{conn: conn}

	if err := c.conn.SetReadDeadline(time.Now().Add(settings.PongWait)); err != nil {
		c.conn.Close()
		return nil, err
	}

	c.conn.SetReadLimit(settings.MaxMessageSize)
	c.conn.SetPingHandler(func(string) error {
		return c.writeMessage(pongMessage)
	})
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(settings.PongWait))
	})

	c.pingPeriod = settings.PingPeriod
	c.writeWait = settings.WriteWait
	c.messageReceived = make(chan Message)
	c.errored = make(chan error, 2)
	c.readPumpFinished = make(chan struct{})
	c.writePumpFinished = make(chan struct{})
	return c, nil
}

// Stream retrieve the peer's message data from the stream channel.
// If the connection closed, this channel closes too.
func (c *Conn) Stream() <-chan Message {
	return c.messageReceived
}

// Err returns the disconnection error if the connection closed.
func (c *Conn) Err() error {
	return c.err
}

// SendBinaryMessage to the peer. This method is goroutine safe.
func (c *Conn) SendBinaryMessage(data []byte) error {
	return c.writeMessage(Message{websocket.BinaryMessage, data})
}

// SendTextMessage to the peer. This method is goroutine safe.
func (c *Conn) SendTextMessage(text string) error {
	return c.writeMessage(Message{websocket.TextMessage, []byte(text)})
}

// SendJSONMessage to the peer. This method is goroutine safe.
func (c *Conn) SendJSONMessage(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.writeMessage(Message{websocket.TextMessage, data})
}

// Close the connection.
func (c *Conn) Close() error {
	if err := c.writeMessage(closeMessage); err != nil {
		return err
	}
	return nil
}

func (c *Conn) start(ctx context.Context) {
	go c.writePump(ctx)
	go c.readPump(ctx)
}

func (c *Conn) writeMessage(m Message) error {
	c.writeMessageMu.Lock()
	defer c.writeMessageMu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeWait)); err != nil {
		return err
	}
	if err := c.conn.WriteMessage(int(m.MessageType), m.Data); err != nil {
		return err
	}
	return nil
}

func (c *Conn) writePump(ctx context.Context) {
	defer c.conn.Close()

	ticker := time.NewTicker(c.pingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			if err := c.writeMessage(closeMessage); err != nil {
				c.errored <- err
				break loop
			}
			c.errored <- ctx.Err()
			break loop
		case <-c.readPumpFinished:
			break loop
		case <-ticker.C:
			if err := c.writeMessage(pingMessage); err != nil {
				c.errored <- err
				break loop
			}
		}
	}
	close(c.writePumpFinished)
}

func (c *Conn) readPump(ctx context.Context) {
	defer c.conn.Close()

loop:
	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			c.errored <- err
			break loop
		}

		var m Message
		switch messageType {
		case websocket.TextMessage:
			m = Message{TextMessageType, data}
		case websocket.BinaryMessage:
			m = Message{BinaryMessageType, data}
		default:
			continue
		}

		select {
		case <-ctx.Done():
			c.errored <- ctx.Err()
			break loop
		case <-c.writePumpFinished:
			break loop
		case c.messageReceived <- m:
		}
	}
	close(c.readPumpFinished)
	<-c.writePumpFinished

	err := <-c.errored
	if err != nil {
		c.err = err
	}
	close(c.messageReceived)
}
