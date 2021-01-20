# websocket-conn [![Build Status](https://secure.travis-ci.org/shiwano/websocket-conn.png?branch=master)](http://travis-ci.org/shiwano/websocket-conn)

> :telephone_receiver: A dead simple WebSocket connection written in Go.

websocket-conn provides you with easy handling for WebSockets, it's based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket).

## Installation

```bash
$ go get -u github.com/shiwano/websocket-conn/v4
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
  SendJSONMessage(v interface{}) error
  Close() error
}
```

## Examples

Server:

```go
package main

import (
  "context"
  "net/http"
  wsconn "github.com/shiwano/websocket-conn/v3"
)

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    c, _ := wsconn.UpgradeFromHTTP(r.Context(), wsconn.DefaultSettings(), w, r)

    m := <-c.Stream()
    c.SendTextMessage(m.Text() + " World")
    m = <-c.Stream()
    if m.Text() == "Close" {
      c.Close()
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
  wsconn "github.com/shiwano/websocket-conn/v3"
)

func main() {
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  c, _, _ := wsconn.Connect(ctx, wsconn.DefaultSettings(), "ws://localhost:5000", nil)

  c.SendTextMessage("Hello")
  m := <-c.Stream()
  log.Println(m.Text()) // Hello World
  c.SendTextMessage("Close")

  for range c.Stream() {
    // wait for closing.
  }
}
```

See also examples directory.

## License

Copyright (c) 2016 Shogo Iwano
Licensed under the MIT license.
