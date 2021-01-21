package main

import (
	"context"
	"log"

	wsconn "github.com/shiwano/websocket-conn/v4"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _, err := wsconn.Connect(ctx, wsconn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.SendTextMessage("Hello"); err != nil {
		log.Fatal(err)
	}

	m := <-c.Stream()
	log.Println(m.Text()) // Hello World

	if err := c.SendTextMessage("Close"); err != nil {
		log.Fatal(err)
	}

	for range c.Stream() {
		// wait for closing.
	}
	log.Println("Closed: ", c.Err())
}
