package conn

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrMessageChannelFull indicates that the connection's message channel is full.
	ErrMessageChannelFull = errors.New("websocket-conn: Message channel is full")
	errEndOfStream        = errors.New("websocket-conn: EOS")

	closeMessage = Message{websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")}
	pingMessage  = Message{websocket.PingMessage, []byte{}}
	pongMessage  = Message{websocket.PongMessage, []byte{}}
)

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
	c := &Conn{conn: conn}
	c.start(ctx, settings)
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
	c := &Conn{conn: conn}
	c.start(ctx, settings)
	return c, nil
}

// Conn represents a WebSocket connection.
type Conn struct {
	conn *websocket.Conn
	err  error

	pingPeriod time.Duration
	writeWait  time.Duration

	streamCh         chan Data
	errorCh          chan error
	sendMessageCh    chan Message
	readPumpFinishCh chan struct{}
}

// Stream retrieve the peer's message data from the stream channel.
// If the connection closed, it returns data with true of EOS flag at last.
func (c *Conn) Stream() <-chan Data {
	return c.streamCh
}

// Err returns the disconnection error if the connection is closed.
func (c *Conn) Err() error {
	return c.err
}

// SendBinaryMessage to the peer. This method is goroutine safe.
func (c *Conn) SendBinaryMessage(data []byte) error {
	return c.sendMessage(Message{websocket.BinaryMessage, data})
}

// SendTextMessage to the peer. This method is goroutine safe.
func (c *Conn) SendTextMessage(text string) error {
	return c.sendMessage(Message{websocket.TextMessage, []byte(text)})
}

func (c *Conn) start(ctx context.Context, settings Settings) {
	c.conn.SetReadLimit(settings.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(settings.PongWait))
	c.conn.SetPingHandler(func(string) error {
		return c.sendMessage(pongMessage)
	})
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(settings.PongWait))
	})

	c.pingPeriod = settings.PingPeriod
	c.writeWait = settings.WriteWait

	c.sendMessageCh = make(chan Message, settings.MessageChannelBufferSize)
	c.streamCh = make(chan Data)
	c.errorCh = make(chan error, 2)
	c.readPumpFinishCh = make(chan struct{}, 1)

	go c.writePump(ctx)
	go c.readPump()
}

func (c *Conn) sendMessage(m Message) error {
	select {
	case c.sendMessageCh <- m:
		return nil
	default:
		return ErrMessageChannelFull
	}
}

func (c *Conn) writeMessage(m Message) error {
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

	for {
		select {
		case <-ctx.Done():
			close(c.sendMessageCh)
			for m := range c.sendMessageCh {
				if err := c.writeMessage(m); err != nil {
					c.errorCh <- err
					return
				}
			}
			if err := c.writeMessage(closeMessage); err != nil {
				c.errorCh <- err
				return
			}
			c.errorCh <- ctx.Err()
			return
		case <-c.readPumpFinishCh:
			return
		case m := <-c.sendMessageCh:
			if err := c.writeMessage(m); err != nil {
				c.errorCh <- err
				return
			}
		case <-ticker.C:
			if err := c.writeMessage(pingMessage); err != nil {
				c.errorCh <- err
				return
			}
		}
	}
}

func (c *Conn) readPump() {
	defer c.conn.Close()

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			c.errorCh <- err
			break
		}
		switch messageType {
		case websocket.TextMessage:
			c.streamCh <- Data{Message: Message{TextMessageType, data}}
		case websocket.BinaryMessage:
			c.streamCh <- Data{Message: Message{BinaryMessageType, data}}
		}
	}

	c.err = <-c.errorCh
	c.readPumpFinishCh <- struct{}{}

	c.streamCh <- Data{EOS: true}
	close(c.streamCh)
}
