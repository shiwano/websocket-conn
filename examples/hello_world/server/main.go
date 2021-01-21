package main

import (
	"log"
	"net/http"

	wsconn "github.com/shiwano/websocket-conn/v4"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := wsconn.UpgradeFromHTTP(r.Context(), wsconn.DefaultSettings(), w, r)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Println("Client connected")

		for m := range c.Stream() {
			switch m.Text() {
			case "Hello":
				if err := c.SendTextMessage(m.Text() + " World"); err != nil {
					log.Fatal(err)
				}
			case "Close":
				c.Close()
			}
		}

		log.Println("Client closed: ", c.Err())
	})
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatal(err)
	}
}
