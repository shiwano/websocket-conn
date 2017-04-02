package main

import (
	"context"
	"fmt"
	"time"

	"github.com/shiwano/websocket-conn"
)

func main() {
	c := conn.New(context.Background())
	c.TextMessageHandler = func(text string) {
		fmt.Println(text)
	}
	if _, err := c.Connect("ws://localhost:5000", nil); err != nil {
		panic(err)
	}
	c.WriteTextMessage("How are you?")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			c.WriteTextMessage(fmt.Sprintf("%v", now))
		}
	}
}
