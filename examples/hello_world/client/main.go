package main

import (
	"context"
	"log"

	wsconn "github.com/shiwano/websocket-conn/v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _, err := wsconn.Connect(ctx, wsconn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}

	c.SendTextMessage("Hello")
	m := <-c.Stream()
	log.Println(m.Text()) // Hello World
	c.SendTextMessage("Close")

	for range c.Stream() {
	}
	log.Println("Closed: ", c.Err())
}
