package protocols

import (
	"fmt"
	"net"
	"time"
)

const (
	bufSize = 2048
)

type UDPHelper struct {
	conn    net.Conn
	timeout time.Duration
}

func (helper *UDPHelper) Initialize(ip string, port uint16, timeout time.Duration) error {
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return err
	}

	helper.conn = conn
	helper.timeout = timeout

	return nil
}

func (helper *UDPHelper) getTimeout() time.Time {
	return time.Now().Add(helper.timeout)
}

func (helper *UDPHelper) Send(data []byte) error {
	err := helper.conn.SetWriteDeadline(helper.getTimeout())
	if err != nil {
		return err
	}

	_, err = helper.conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (helper *UDPHelper) Receive() (Packet, error) {
	err := helper.conn.SetReadDeadline(helper.getTimeout())
	if err != nil {
		return Packet{}, err
	}

	recvBuffer := make([]byte, bufSize)
	recvSize, err := helper.conn.Read(recvBuffer)
	if err != nil {
		return Packet{}, err
	}

	packet := Packet{}
	packet.SetBuffer(recvBuffer[:recvSize])

	return packet, nil
}

func (helper *UDPHelper) Close() error {
	return helper.conn.Close()
}
