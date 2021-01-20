package wsconn_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shiwano/websocket-conn/v4"
)

type jsonMessage struct {
	Message string `json:"message"`
}

func newTestServer(handler func(http.ResponseWriter, *http.Request)) (*httptest.Server, string) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	url := strings.Replace(ts.URL, "http", "ws", 1)
	return ts, url
}

func TestConn(t *testing.T) {
	serverConnCh := make(chan error)
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rootCtxCancel()

	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsconn.UpgradeFromHTTP(rootCtx, wsconn.DefaultSettings(), w, r)
		if err != nil {
			t.Fatalf("failed to upgrade http connection: %v", err)
		}

		m := <-c.Stream()
		if err := c.SendTextMessage(m.Text() + " PONG"); err != nil {
			t.Fatalf("failed to send a text message: %v", err)
		}

		m = <-c.Stream()
		if err := c.SendBinaryMessage(append(m.Data, 4, 5, 6)); err != nil {
			t.Fatalf("failed to send a binary message: %v", err)
		}

		m = <-c.Stream()
		var msg jsonMessage
		if err := m.UnmarshalAsJSON(&msg); err != nil {
			t.Fatalf("failed to unmarshal message as JSON: %v", err)
		}
		msg.Message += "bar"
		if err := c.SendJSONMessage(&msg); err != nil {
			t.Fatalf("failed to send a JSON message: %v", err)
		}

		for m := range c.Stream() {
			t.Fatalf("received an unexpected data to the server connetion: %v", m)
		}
		serverConnCh <- c.Err()
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(rootCtx)
	c, _, err := wsconn.Connect(ctx, wsconn.DefaultSettings(), url, nil)
	if err != nil {
		t.Fatalf("failed to connect to the server: %v", err)
	}

	if err := c.SendTextMessage("PING"); err != nil {
		t.Fatalf("failed to send a text message: %v", err)
	}

	m := <-c.Stream()
	if !m.IsTextMessage() || m.Text() != "PING PONG" {
		t.Errorf("failed to process a text message: %v", m)
	}

	if err := c.SendBinaryMessage([]byte{1, 2, 3}); err != nil {
		t.Fatalf("failed to send a binary message: %v", err)
	}

	m = <-c.Stream()
	if !m.IsBinaryMessage() || !bytes.Equal(m.Data, []byte{1, 2, 3, 4, 5, 6}) {
		t.Errorf("failed to process a binary message: %v", m)
	}

	if err := c.SendJSONMessage(jsonMessage{Message: "foo"}); err != nil {
		t.Fatalf("failed to send a JSON message: %v", err)
	}

	m = <-c.Stream()
	if !m.IsTextMessage() {
		t.Errorf("failed to process a text message as JSON: %v", m)
	}
	var msg jsonMessage
	if err := m.UnmarshalAsJSON(&msg); err != nil {
		t.Errorf("failed to unmarshal message as JSON: %v", err)
	}
	if msg.Message != "foobar" {
		t.Errorf("want foobar, but got: %s", msg.Message)
	}

	cancel()

	serverConnErr := <-serverConnCh
	if serverConnErr.(*websocket.CloseError).Code != websocket.CloseNormalClosure {
		t.Errorf("unexpected server connection error: %v", serverConnErr)
	}

	for m := range c.Stream() {
		t.Errorf("received an unexpected data to the client connetion: %v", m)
	}

	if c.Err() != context.Canceled {
		t.Errorf("Unexpected client connection error: %v", c.Err())
	}
}

func TestConn_Close_Client(t *testing.T) {
	serverConnCh := make(chan error)
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rootCtxCancel()

	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsconn.UpgradeFromHTTP(rootCtx, wsconn.DefaultSettings(), w, r)
		if err != nil {
			t.Fatalf("failed to upgrade http connection: %v", err)
		}

		for m := range c.Stream() {
			t.Fatalf("received an unexpected data to the server connetion: %v", m)
		}
		serverConnCh <- c.Err()
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	c, _, err := wsconn.Connect(ctx, wsconn.DefaultSettings(), url, nil)
	if err != nil {
		t.Fatalf("failed to connect to the server: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	serverConnErr := <-serverConnCh
	if serverConnErr.(*websocket.CloseError).Code != websocket.CloseNormalClosure {
		t.Errorf("unexpected server connection error: %v", serverConnErr)
	}

	for range c.Stream() {
	}

	if err := c.SendTextMessage("must fail"); err != wsconn.ErrMessageSendingFailed {
		t.Errorf("message sending must fail: %v", err)
	}
	if err := c.SendBinaryMessage([]byte{0}); err != wsconn.ErrMessageSendingFailed {
		t.Errorf("message sending must fail: %v", err)
	}
	if err := c.SendJSONMessage("must fail"); err != wsconn.ErrMessageSendingFailed {
		t.Errorf("message sending must fail: %v", err)
	}
}

func TestConn_Close_Server(t *testing.T) {
	rootCtx, rootCtxCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer rootCtxCancel()

	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsconn.UpgradeFromHTTP(rootCtx, wsconn.DefaultSettings(), w, r)
		if err != nil {
			t.Fatalf("failed to upgrade http connection: %v", err)
		}

		if err := c.Close(); err != nil {
			t.Fatalf("failed to close: %v", err)
		}
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	c, _, err := wsconn.Connect(ctx, wsconn.DefaultSettings(), url, nil)
	if err != nil {
		t.Fatalf("failed to connect to the server: %v", err)
	}

	for m := range c.Stream() {
		t.Errorf("received an unexpected data to the client connetion: %v", m)
	}
}
