package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shiwano/websocket-conn"
)

func main() {
	c, _, err := conn.Connect(context.Background(), conn.DefaultSettings(), "ws://localhost:5000", nil)
	if err != nil {
		log.Fatal(err)
	}
	c.SendTextMessage("How are you?")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case d := <-c.Stream():
			if d.EOS {
				log.Fatal(c.Err())
			}
			log.Println(d.Message.Text())
		case now := <-ticker.C:
			c.SendTextMessage(fmt.Sprintf("%v", now))
		}
	}
}
