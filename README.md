# websocket-conn [![Build Status](https://secure.travis-ci.org/shiwano/websocket-conn.png?branch=master)](http://travis-ci.org/shiwano/websocket-conn)

> :telephone_receiver: A dead simple WebSocket connection written in Go.

websocket-conn provides you with easy handling for WebSockets, it's based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket).

## Installation

```bash
$ go get -u github.com/shiwano/websocket-conn
```

## Usage

```go
func Connect(ctx context.Context, settings Settings, url string, requestHeader http.Header) (*Conn *http.Response, error)
func UpgradeFromHTTP(ctx context.Context, settings Settings, w http.ResponseWriter, r *http.Request) (*Conn, error)

type Conn struct {
	Stream() <-chan Data
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
		d := <-c.Stream()
		c.SendTextMessage(d.Message.Text() + " World")
		cancel()
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
	d := <-c.Stream()
	log.Println(d.Message.Text()) // Hello World
	for d := range c.Stream() {
		if d.EOS {
			log.Println("Closed: ", c.Err())
		}
	}
}
```

See also examples directory.

## License

Copyright (c) 2016 Shogo Iwano
Licensed under the MIT license.
