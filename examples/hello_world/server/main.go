package main

import (
	"context"
	"log"
	"net/http"

	wsconn "github.com/shiwano/websocket-conn/v3"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c, err := wsconn.UpgradeFromHTTP(ctx, wsconn.DefaultSettings(), w, r)
		if err != nil {
			w.Write([]byte("Error"))
			return
		}
		log.Println("Client connected")
		m := <-c.Stream()
		c.SendTextMessage(m.Text() + " World")
		cancel()
		for range c.Stream() {
		}
		log.Println("Client closed: ", c.Err())
	})
	http.ListenAndServe(":5000", nil)
}
