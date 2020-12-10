package main

import (
	"context"
	"log"

	wsconn "github.com/shiwano/websocket-conn"
)

func main() {
	c, _, err := wsconn.Connect(context.Background(), wsconn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}
	c.SendTextMessage("Hello")
	m := <-c.Stream()
	log.Println(m.Text()) // Hello World
	for range c.Stream() {
	}
	log.Println("Closed: ", c.Err())
}
