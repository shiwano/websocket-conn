package conn

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func newTestServer(handler func(http.ResponseWriter, *http.Request)) (*httptest.Server, string) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	url := strings.Replace(ts.URL, "http", "ws", 1)
	return ts, url
}

func TestConn(t *testing.T) {
	serverDisconnectionCh := make(chan error)
	clientDisconnectionCh := make(chan error)
	textMessageCh := make(chan string)
	binaryMessageCh := make(chan []byte)
	rootCtx := context.Background()

	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c2 := New(rootCtx)
		c2.TextMessageHandler = func(t string) { c2.WriteTextMessage(t + " PONG") }
		c2.BinaryMessageHandler = func(d []byte) { c2.WriteBinaryMessage(append(d, 4, 5, 6)) }
		c2.DisconnectionHandler = func(err error) { serverDisconnectionCh <- err }

		if err := c2.UpgradeFromHTTP(w, r); err != nil {
			t.Error(err)
		}
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(rootCtx)
	c := New(ctx)
	c.TextMessageHandler = func(t string) { textMessageCh <- t }
	c.BinaryMessageHandler = func(d []byte) { binaryMessageCh <- d }
	c.DisconnectionHandler = func(err error) { clientDisconnectionCh <- err }

	if _, err := c.Connect(url, nil); err != nil {
		t.Error(err)
	}

	c.WriteTextMessage("PING")
	textMessage := <-textMessageCh
	if textMessage != "PING PONG" {
		t.Error(fmt.Errorf("Failed to send or receive a text message: %v", textMessage))
	}

	c.WriteBinaryMessage([]byte{1, 2, 3})
	binaryMessage := <-binaryMessageCh
	if !bytes.Equal(binaryMessage, []byte{1, 2, 3, 4, 5, 6}) {
		t.Error(fmt.Errorf("Failed to send or receive a binary message: %v", binaryMessage))
	}

	cancel()

	clientErr := <-clientDisconnectionCh
	if clientErr.Error() != "context canceled" {
		t.Error(fmt.Errorf("Unexpected client connection error: %v", clientErr))
	}

	serverErr := <-serverDisconnectionCh
	if serverErr.(*websocket.CloseError).Code != websocket.CloseNormalClosure {
		t.Error(fmt.Errorf("Unexpected server connection error: %v", clientErr))
	}
}
