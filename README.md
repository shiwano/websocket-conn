# websocket-conn [![Build Status](https://secure.travis-ci.org/shiwano/websocket-conn.png?branch=master)](http://travis-ci.org/shiwano/websocket-conn)

A dead simple WebSocket connection written in Go.

## Installation

```bash
$ go get github.com/shiwano/websocket-conn
```

## Usage

Server:

```go
package main

import (
  "github.com/shiwano/websocket-conn"
  "net/http"
)

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    c := conn.New()
    c.TextMessageHandler = func(text string) {
      c.WriteTextMessage(text + " World")
    }
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
  "fmt"
  "github.com/shiwano/websocket-conn"
  "time"
)

func main() {
  c := conn.New()
  textMessageCh := make(chan string)
  c.TextMessageHandler = func(text string) {
    textMessageCh <- text
  }
  if _, err := c.Connect("ws://localhost:5000", nil); err != nil {
    panic(err)
  }
  c.WriteTextMessage("Hello")
  text := <-textMessageCh
  fmt.Println(text)
}
```

## License

Copyright (c) 2016 Shogo Iwano
Licensed under the MIT license.
