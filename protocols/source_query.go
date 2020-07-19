package protocols

type SourceQuery struct{}

func (sq SourceQuery) Names() []string {
	return []string{
		"source",
	}
}

func (sq SourceQuery) DefaultPort() uint16 {
	return 27015
}

func (sq SourceQuery) Helper() string {
	return "udp"
}

func (sq SourceQuery) Execute(helper NetworkHelper) (Response, error) {
	packet := Packet{}
	packet.WriteRaw(0xFF, 0xFF, 0xFF, 0xFF, 0x54)
	packet.WriteString("Source Engine Query")

	// TODO: Send packet & receive the data

	return Response{}, nil
}
