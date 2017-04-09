package main

import (
	"context"
	"log"
	"net/http"

	"github.com/shiwano/websocket-conn"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c, err := conn.UpgradeFromHTTP(ctx, conn.DefaultSettings(), w, r)
		if err != nil {
			w.Write([]byte("Error"))
			return
		}
		log.Println("Client connected")
		d := <-c.Stream()
		c.SendTextMessage(d.Message.Text() + " World")
		cancel()
		for d := range c.Stream() {
			if d.EOS {
				log.Println("Client closed: ", c.Err())
			}
		}
	})
	http.ListenAndServe(":5000", nil)
}
