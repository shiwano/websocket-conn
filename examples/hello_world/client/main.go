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
