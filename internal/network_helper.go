package internal

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	readBufSize = 2048
)

type NetworkHelper struct {
	ip      string
	port    uint16
	conn    net.Conn
	timeout time.Duration
}

func (helper *NetworkHelper) Initialize(protocol string, ip string, port uint16, timeout time.Duration) error {
	conn, err := net.DialTimeout(protocol, fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return err
	}

	helper.ip = ip
	helper.port = port
	helper.conn = conn
	helper.timeout = timeout

	return nil
}

func (helper *NetworkHelper) getTimeout() time.Time {
	return time.Now().Add(helper.timeout)
}

func (helper *NetworkHelper) Send(data []byte) error {
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

func (helper *NetworkHelper) Receive() (Packet, error) {
	err := helper.conn.SetReadDeadline(helper.getTimeout())
	if err != nil {
		return Packet{}, err
	}

	var res = &bytes.Buffer{}
	for {
		recvBuffer := make([]byte, readBufSize)
		recvSize, err := helper.conn.Read(recvBuffer)

		if recvSize > 0 {
			res.Write(recvBuffer[:recvSize])
		}

		if err == io.EOF || recvSize < readBufSize {
			break
		}

		if err != nil {
			return Packet{}, err
		}
	}

	packet := Packet{}
	packet.SetBuffer(res.Bytes())

	return packet, nil
}

func (helper *NetworkHelper) Close() error {
	return helper.conn.Close()
}

func (helper *NetworkHelper) GetIP() string {
	return helper.ip
}

func (helper *NetworkHelper) GetPort() uint16 {
	return helper.port
}
