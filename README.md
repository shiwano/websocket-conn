# websocket-conn [![Build Status](https://secure.travis-ci.org/shiwano/websocket-conn.png?branch=master)](http://travis-ci.org/shiwano/websocket-conn)

> :telephone_receiver: A dead simple WebSocket connection written in Go.

websocket-conn provides you with easy handling for WebSockets, it's based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket).

## Installation

```bash
$ go get -u github.com/shiwano/websocket-conn/v3
```

## Usage

```go
func Connect(ctx context.Context, settings Settings, url string, requestHeader http.Header) (*Conn *http.Response, error)
func UpgradeFromHTTP(ctx context.Context, settings Settings, w http.ResponseWriter, r *http.Request) (*Conn, error)

type Conn struct {
	Stream() <-chan Message
	Err() error
	SendBinaryMessage(data []byte) error
	SendTextMessage(text string) error
}
```

## Examples

Server:

```go
package main

import (
	"context"
	"net/http"
	"github.com/shiwano/websocket-conn"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c, err := conn.UpgradeFromHTTP(ctx, conn.DefaultSettings(), w, r)
		if err != nil {
			w.Write([]byte("Error"))
			return
		}
		m := <-c.Stream()
		c.SendTextMessage(m.Text() + " World")
		cancel()
		for range c.Stream() {
		}
	})
	http.ListenAndServe(":5000", nil)
}
```

Client:

```go
package main

import (
	"context"
	"log"
	"github.com/shiwano/websocket-conn"
)

func main() {
	c, _, err := conn.Connect(context.Background(), conn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}
	c.SendTextMessage("Hello")
	m := <-c.Stream()
	log.Println(m.Text()) // Hello World
	for range c.Stream() {
	}
}
```

See also examples directory.

## License

Copyright (c) 2016 Shogo Iwano
Licensed under the MIT license.
