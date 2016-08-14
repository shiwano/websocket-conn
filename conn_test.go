package conn

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler func(http.ResponseWriter, *http.Request)) (*httptest.Server, string) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	url := strings.Replace(ts.URL, "http", "ws", 1)
	return ts, url
}

func TestConn(t *testing.T) {
	ts, url := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		c2 := New()
		c2.TextMessageHandler = func(text string) {
			c2.WriteTextMessage(text + " PONG")
		}
		c2.BinaryMessageHandler = func(data []byte) {
			c2.WriteBinaryMessage(append(data, 4, 5, 6))
		}

		if err := c2.UpgradeFromHTTP(w, r); err != nil {
			t.Error(err)
		}
	})
	defer ts.Close()

	textMessageCh := make(chan string)
	binaryMessageCh := make(chan []byte)
	disconnectedCh := make(chan struct{})

	c := New()
	c.TextMessageHandler = func(t string) { textMessageCh <- t }
	c.BinaryMessageHandler = func(d []byte) { binaryMessageCh <- d }
	c.DisconnectHandler = func() { disconnectedCh <- struct{}{} }

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

	c.Close()
	<-disconnectedCh
}
