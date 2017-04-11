package main

import (
	"context"
	"log"
	"net/http"

	"github.com/shiwano/websocket-conn"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("client arrived")
		c, err := conn.UpgradeFromHTTP(context.Background(), conn.DefaultSettings(), w, r)
		if err != nil {
			w.Write([]byte("Error"))
			return
		}
		for m := range c.Stream() {
			if m.IsTextMessage() {
				text := m.Text()

				if text == "How are you?" {
					c.SendTextMessage("I'm fine, thank you")
				} else {
					c.SendTextMessage("Ticktack: " + text)
				}
			}
		}
		log.Println("client left because of: ", c.Err())
	})
	http.ListenAndServe(":5000", nil)
}
