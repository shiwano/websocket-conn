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

		m := <-c.Stream()
		c.SendTextMessage(m.Text() + " PONG")
		m = <-c.Stream()
		c.SendBinaryMessage(append(m.Data, 4, 5, 6))

		for m := range c.Stream() {
			t.Error(fmt.Errorf("Received an unexpected data to the server connetion: %v", m))
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
	m := <-c.Stream()
	if !m.IsTextMessage() || m.Text() != "PING PONG" {
		t.Error(fmt.Errorf("Failed to process a text message: %v", m))
	}

	c.SendBinaryMessage([]byte{1, 2, 3})
	m = <-c.Stream()
	if !m.IsBinaryMessage() || !bytes.Equal(m.Data, []byte{1, 2, 3, 4, 5, 6}) {
		t.Error(fmt.Errorf("Failed to process a binary message: %v", m))
	}

	cancel()

	serverConnErr := <-serverConnCh
	if serverConnErr.(*websocket.CloseError).Code != websocket.CloseNormalClosure {
		t.Error(fmt.Errorf("Unexpected server connection error: %v", serverConnErr))
	}

	for m := range c.Stream() {
		t.Error(fmt.Errorf("Received an unexpected data to the client connetion: %v", m))
	}
	if c.Err() != context.Canceled {
		t.Error(fmt.Errorf("Unexpected client connection error: %v", c.Err()))
	}
}
