package conn_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/shiwano/websocket-conn"
)

func newTestServer(handler func(http.ResponseWriter, *http.Request)) (*httptest.Server, string) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	url := strings.Replace(ts.URL, "http", "ws", 1)
	return ts, url
}

func TestConn(t *testing.T) {
	serverConnCh := make(chan error)
	rootCtx := context.Background()

	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c, err := conn.UpgradeFromHTTP(rootCtx, conn.DefaultSettings(), w, r)
		if err != nil {
			t.Error(err)
		}

		d := <-c.Stream()
		c.SendTextMessage(d.Message.Text() + " PONG")
		d = <-c.Stream()
		c.SendBinaryMessage(append(d.Message.Data, 4, 5, 6))

		for d := range c.Stream() {
			if !d.EOS {
				t.Error(fmt.Errorf("Received an unexpected data to the server connetion: %v", d))
			}
		}
		serverConnCh <- c.Err()
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(rootCtx)
	c, _, err := conn.Connect(ctx, conn.DefaultSettings(), url, nil)
	if err != nil {
		t.Error(err)
	}

	c.SendTextMessage("PING")
	d := <-c.Stream()
	if d.Message.MessageType != conn.TextMessageType || d.Message.Text() != "PING PONG" {
		t.Error(fmt.Errorf("Failed to process a text message: %v", d))
	}

	c.SendBinaryMessage([]byte{1, 2, 3})
	d = <-c.Stream()
	if d.Message.MessageType != conn.BinaryMessageType || !bytes.Equal(d.Message.Data, []byte{1, 2, 3, 4, 5, 6}) {
		t.Error(fmt.Errorf("Failed to process a binary message: %v", d))
	}

	cancel()

	serverConnErr := <-serverConnCh
	if serverConnErr.(*websocket.CloseError).Code != websocket.CloseNormalClosure {
		t.Error(fmt.Errorf("Unexpected server connection error: %v", serverConnErr))
	}

	for d := range c.Stream() {
		if !d.EOS {
			t.Error(fmt.Errorf("Received an unexpected data to the client connetion: %v", d))
		}
	}
	if c.Err().Error() != "context canceled" {
		t.Error(fmt.Errorf("Unexpected client connection error: %v", c.Err()))
	}
}
