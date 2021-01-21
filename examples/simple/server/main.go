package main

import (
	"context"
	"log"
	"net/http"

	wsconn "github.com/shiwano/websocket-conn/v4"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("client arrived")
		c, err := wsconn.UpgradeFromHTTP(context.Background(), wsconn.DefaultSettings(), w, r)
		if err != nil {
			log.Fatal(err)
			return
		}
		for m := range c.Stream() {
			if m.IsTextMessage() {
				text := m.Text()

				if text == "How are you?" {
					if err := c.SendTextMessage("I'm fine, thank you"); err != nil {
						log.Fatal(err)
					}
				} else {
					if err := c.SendTextMessage("Ticktack: " + text); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
		log.Println("client left because of: ", c.Err())
	})
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatal(err)
	}
}
