package protocols

import (
	"encoding/binary"
)

type Packet struct {
	buffer []byte
	pos    int

	order binary.ByteOrder
}

func (p *Packet) WriteRaw(bytes ...byte) {
	for _, b := range bytes {
		p.buffer = append(p.buffer, b)
	}
}

func (p *Packet) WriteInt32(int int32) {
	buf := make([]byte, 4)
	p.order.PutUint32(buf, uint32(int))

	p.WriteRaw(buf...)
}

func (p *Packet) WriteUint8(int uint8) {
	p.WriteRaw(int)
}

func (p *Packet) WriteUint16(int uint16) {
	buf := make([]byte, 2)
	p.order.PutUint16(buf, uint16(int))

	p.WriteRaw(buf...)
}

func (p *Packet) WriteVarint(num int) {
	res := make([]byte, 0)
	for {
		b := num & 0x7F
		num >>= 7

		if num != 0 {
			b |= 0x80
		}

		res = append(res, byte(b))

		if num == 0 {
			break
		}
	}

	p.WriteRaw(res...)
}

func (p *Packet) WriteString(str string) {
	p.WriteRaw([]byte(str)...)
}

func (p *Packet) ReadInt32() int32 {
	r := int32(p.order.Uint32(p.buffer[p.pos : p.pos+4]))
	p.pos += 4

	return r
}

func (p *Packet) ReadUint8() uint8 {
	r := p.buffer[p.pos]
	p.pos++

	return r
}

func (p *Packet) ReadUint16() uint16 {
	r := p.order.Uint16(p.buffer[p.pos : p.pos+2])
	p.pos += 2

	return r
}

func (p *Packet) ReadUint64() uint64 {
	r := p.order.Uint64(p.buffer[p.pos : p.pos+8])
	p.pos += 8

	return r
}

func (p *Packet) ReadVarint() int {
	var varint = 0
	for i := 0; i <= 5; i++ {
		nextByte := p.ReadUint8()
		varint |= (int(nextByte) & 0x7F) << (7 * i)

		if (nextByte & 0x80) == 0 {
			break
		}
	}

	return varint
}

func (p *Packet) ReadString() string {
	start := p.pos
	for {
		if p.ReachedEnd() || p.buffer[p.pos] == 0x00 {
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

func (p *Packet) SetOrder(order binary.ByteOrder) {
	p.order = order
}

func (p *Packet) SetBuffer(buffer []byte) {
	p.buffer = buffer
}

func (p *Packet) GetBuffer() []byte {
	return p.buffer
}

func (p *Packet) Forward(count int) {
	p.pos += count
}

func (p *Packet) Clear() {
	p.pos = 0
	p.buffer = make([]byte, 0)
}

func (p *Packet) Length() int {
	return len(p.buffer)
}

func (p *Packet) AsString() string {
	return string(p.GetBuffer())
}
