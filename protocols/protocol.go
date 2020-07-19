package protocols

type Response struct {
	// TODO: Decide the structure of this

	Raw interface{}
}

type Protocol interface {
	Names() []string
	DefaultPort() uint16
	Priority() uint16
	Network() string

	Execute(helper NetworkHelper) (Response, error)
}
