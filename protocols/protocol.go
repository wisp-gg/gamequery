package protocols

import "time"

type Request struct {
	Game    string
	IP      string
	Port    *uint16
	Timeout *time.Duration
}

type Response struct {
	// TODO: Decide the structure of this

	Raw interface{}
}

type NetworkHelper interface {
	Initialize(ip string, port uint16, timeout time.Duration) error
	Send(data []byte) error
	Receive(size uint16) (Packet, error)
	Close() error
}

type Protocol interface {
	Names() []string
	DefaultPort() uint16
	Helper() string

	Execute(helper NetworkHelper) (Response, error)
}
