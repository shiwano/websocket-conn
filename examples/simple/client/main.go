package main

import (
	"context"
	"fmt"
	"log"
	"time"

	wsconn "github.com/shiwano/websocket-conn/v4"
)

func main() {
	c, _, err := wsconn.Connect(context.Background(), wsconn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.SendTextMessage("How are you?"); err != nil {
		log.Fatal(err)
	}

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
			if err := c.SendTextMessage(fmt.Sprintf("%v", now)); err != nil {
				log.Fatal(err)
			}
		}
	}
}
