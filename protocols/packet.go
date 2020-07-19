package protocols

import "encoding/binary"

type Packet struct {
	buffer []byte
	pos    int
}

func (p *Packet) WriteRaw(bytes ...byte) {
	for _, b := range bytes {
		p.buffer = append(p.buffer, b)
	}
}

func (p *Packet) WriteString(str string) {
	p.WriteRaw([]byte(str)...)
}

func (p *Packet) ReadInt32() int32 {
	r := int32(binary.LittleEndian.Uint32(p.buffer[p.pos : p.pos+4]))
	p.pos += 4

	return r
}

func (p *Packet) ReadUint8() uint8 {
	r := p.buffer[p.pos]
	p.pos++

	return r
}

func (p *Packet) ReadUint16() uint16 {
	r := binary.LittleEndian.Uint16(p.buffer[p.pos : p.pos+2])
	p.pos += 2

	return r
}

func (p *Packet) ReadUint64() uint64 {
	r := binary.LittleEndian.Uint64(p.buffer[p.pos : p.pos+8])
	p.pos += 8

	return r
}

func (p *Packet) ReadString() string {
	start := p.pos
	for {
		if p.buffer[p.pos] == 0x00 {
			break
		}

		p.pos++
	}

	str := p.buffer[start:p.pos]
	p.pos++

	return string(str)
}

func (p *Packet) ReachedEnd() bool {
	return p.pos >= len(p.buffer)
}

func (p *Packet) SetBuffer(buffer []byte) {
	p.buffer = buffer
}

func (p *Packet) GetBuffer() []byte {
	return p.buffer
}

func (p *Packet) AsString() string {
	return string(p.GetBuffer())
}
