package wsconn

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Settings represents connection settings.
type Settings struct {
	WriteWait        time.Duration
	PongWait         time.Duration
	PingPeriod       time.Duration
	HandshakeTimeout time.Duration
	MaxMessageSize   int64
	ReadBufferSize   int
	WriteBufferSize  int
	Subprotocols     []string

	DialerSettings   *DialerSettings
	UpgraderSettings *UpgraderSettings
}

// DialerSettings represents websocket.Dialer settings.
type DialerSettings struct {
	NetDial         func(network, addr string) (net.Conn, error)
	TLSClientConfig *tls.Config
}

// UpgraderSettings represents websocket.Upgrader settings.
type UpgraderSettings struct {
	Error       func(w http.ResponseWriter, r *http.Request, status int, reason error)
	CheckOrigin func(r *http.Request) bool
}

// DefaultSettings returns default settings.
func DefaultSettings() Settings {
	return Settings{
		WriteWait:       10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      54 * time.Second,
		MaxMessageSize:  2048,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,

		DialerSettings:   new(DialerSettings),
		UpgraderSettings: new(UpgraderSettings),
	}
}
