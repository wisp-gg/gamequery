package protocols

type Packet struct {
	buffer []byte
}

func (p *Packet) SetBuffer(buffer []byte) {
	p.buffer = buffer
}

func (p *Packet) GetBuffer() []byte {
	return p.buffer
}

func (p *Packet) WriteRaw(bytes ...byte) {
	for _, b := range bytes {
		p.buffer = append(p.buffer, b)
	}
}

func (p *Packet) WriteString(str string) {
	p.WriteRaw([]byte(str)...)
}

func (p *Packet) AsString() string {
	return string(p.GetBuffer())
}
