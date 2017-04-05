# websocket-conn [![Build Status](https://secure.travis-ci.org/shiwano/websocket-conn.png?branch=master)](http://travis-ci.org/shiwano/websocket-conn)

> :telephone_receiver: A dead simple WebSocket connection written in Go.

websocket-conn provides you with easy handling for WebSockets, it's based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket).

## Installation

```bash
$ go get -u github.com/shiwano/websocket-conn
```

## Usage

```go
type Conn struct {
  Settings             *Settings
  BinaryMessageHandler func([]byte)
  TextMessageHandler   func(string)
  DisconnectionHandler func(error)
}

func New(ctx context.Context) *Conn
func (c *Conn) Connect(url string, requestHeader http.Header) (*http.Response, error)
func (c *Conn) UpgradeFromHTTP(responseWriter http.ResponseWriter, request *http.Request) error
func (c *Conn) WriteBinaryMessage(data []byte) error
func (c *Conn) WriteTextMessage(text string) error
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
    ctx := context.Background()
    c := conn.New(ctx)
    c.TextMessageHandler = func(text string) { c.WriteTextMessage(text + " World") }
    if err := c.UpgradeFromHTTP(w, r); err != nil {
      w.Write([]byte("Error"))
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
  "fmt"
  "github.com/shiwano/websocket-conn"
)

func main() {
  textMessageCh := make(chan string)
  ctx := context.Background()
  c := conn.New(ctx)
  c.TextMessageHandler = func(text string) { textMessageCh <- text }
  if _, err := c.Connect("ws://localhost:5000", nil); err != nil {
    panic(err)
  }
  c.WriteTextMessage("Hello")
  text := <-textMessageCh
  fmt.Println(text) // Hello World
}
```

## License

Copyright (c) 2016 Shogo Iwano
Licensed under the MIT license.
