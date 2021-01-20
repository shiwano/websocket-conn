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
			w.Write([]byte("Error"))
			return
		}
		log.Println("Client connected")

		m := <-c.Stream()
		c.SendTextMessage(m.Text() + " World")
		m = <-c.Stream()
		if m.Text() == "Close" {
			c.Close()
		}

		log.Println("Client closed: ", c.Err())
	})
	http.ListenAndServe(":5000", nil)
}
