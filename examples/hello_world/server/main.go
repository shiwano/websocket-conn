package main

import (
	"context"
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
		d := <-c.Stream()
		c.SendTextMessage(d.Message.Text() + " World")
		cancel()
	})
	http.ListenAndServe(":5000", nil)
}
