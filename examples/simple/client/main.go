package main

import (
	"context"
	"fmt"
	"log"
	"time"

	wsconn "github.com/shiwano/websocket-conn/v3"
)

func main() {
	c, _, err := wsconn.Connect(context.Background(), wsconn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}
	c.SendTextMessage("How are you?")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case m, ok := <-c.Stream():
			if !ok {
				log.Fatal(c.Err())
			}
			if m.IsTextMessage() {
				log.Println(m.Text())
			}
		case now := <-ticker.C:
			c.SendTextMessage(fmt.Sprintf("%v", now))
		}
	}
}
