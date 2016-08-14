package conn

import (
	"time"
)

// Settings represents connection settings.
type Settings struct {
	WriteWait                time.Duration
	PongWait                 time.Duration
	PingPeriod               time.Duration
	HandshakeTimeout         time.Duration
	MessageChannelBufferSize int
	MaxMessageSize           int64
	ReadBufferSize           int
	WriteBufferSize          int
}

// newDefaultSettings creates default settings.
func newDefaultSettings() *Settings {
	return &Settings{
		WriteWait:                10 * time.Second,
		PongWait:                 60 * time.Second,
		PingPeriod:               54 * time.Second,
		MessageChannelBufferSize: 256,
		MaxMessageSize:           2048,
		ReadBufferSize:           4096,
		WriteBufferSize:          4096,
	}
}
