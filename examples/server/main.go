package main

import (
	"context"
	"net/http"

	"github.com/shiwano/websocket-conn"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c := conn.New(context.Background())
		c.TextMessageHandler = func(text string) {
			if text == "How are you?" {
				c.WriteTextMessage("I'm fine, thank you")
			} else {
				c.WriteTextMessage("Ticktack: " + text)
			}
		}
		if err := c.UpgradeFromHTTP(w, r); err != nil {
			w.Write([]byte("Error"))
		}
	})
	http.ListenAndServe(":5000", nil)
}
