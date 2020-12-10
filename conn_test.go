package conn_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	conn "github.com/shiwano/websocket-conn"
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
		c, err := conn.UpgradeFromHTTP(rootCtx, conn.DefaultSettings(), w, r)
		if err != nil {
			t.Error(err)
		}

		m := <-c.Stream()
		if err := c.SendTextMessage(m.Text() + " PONG"); err != nil {
			t.Fatal(err)
		}

		m = <-c.Stream()
		if err := c.SendBinaryMessage(append(m.Data, 4, 5, 6)); err != nil {
			t.Fatal(err)
		}

		m = <-c.Stream()
		var msg jsonMessage
		if err := m.UnmarshalAsJSON(&msg); err != nil {
			t.Fatal(err)
		}
		msg.Message += "bar"
		if err := c.SendJSONMessage(&msg); err != nil {
			t.Fatal(err)
		}

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

	if err := c.SendTextMessage("PING"); err != nil {
		t.Fatal(err)
	}

	m := <-c.Stream()
	if !m.IsTextMessage() || m.Text() != "PING PONG" {
		t.Error(fmt.Errorf("Failed to process a text message: %v", m))
	}

	if err := c.SendBinaryMessage([]byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	m = <-c.Stream()
	if !m.IsBinaryMessage() || !bytes.Equal(m.Data, []byte{1, 2, 3, 4, 5, 6}) {
		t.Error(fmt.Errorf("Failed to process a binary message: %v", m))
	}

	if err := c.SendJSONMessage(jsonMessage{Message: "foo"}); err != nil {
		t.Fatal(err)
	}

	m = <-c.Stream()
	if !m.IsTextMessage() {
		t.Error(fmt.Errorf("Failed to process a text message as JSON: %v", m))
	}
	var msg jsonMessage
	if err := m.UnmarshalAsJSON(&msg); err != nil {
		t.Error(err)
	}
	if msg.Message != "foobar" {
		t.Errorf("want foobar, but got: %s", msg.Message)
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
