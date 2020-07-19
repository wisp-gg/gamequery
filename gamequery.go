package gamequery

import (
	"errors"
	"github.com/wisp-gg/gamequery/protocols"
	"reflect"
	"time"
)

type Request struct {
	Game    string
	IP      string
	Port    uint16
	Timeout *time.Duration
}

var helpers = map[string]reflect.Type{
	"udp": reflect.TypeOf(protocols.UDPHelper{}),
}

var queryProtocols = []protocols.Protocol{
	protocols.SourceQuery{},
}

func findProtocol(name string) (protocols.Protocol, error) {
	for _, protocol := range queryProtocols { // TODO: Optimize this to a lookup list?
		for _, protocolName := range protocol.Names() {
			if protocolName == name {
				return protocol, nil
			}
		}
	}

	return nil, errors.New("could not find protocol for the game") // TODO: Should be easily checkable
}

func Query(req Request) (protocols.Response, error) {
	queryProtocol, err := findProtocol(req.Game)
	if err != nil {
		return protocols.Response{}, err
	}

	var port = queryProtocol.DefaultPort()
	if req.Port != 0 {
		port = req.Port
	}

	var timeout = 5 * time.Second
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	networkType := helpers[queryProtocol.Helper()]
	if networkType == nil {
		return protocols.Response{}, errors.New("unknown helper required for requested protocol")
	}

	networkHelper := reflect.New(networkType).Interface().(protocols.NetworkHelper)
	if err := networkHelper.Initialize(req.IP, port, timeout); err != nil {
		return protocols.Response{}, err
	}

	response, err := queryProtocol.Execute(networkHelper)
	if err != nil {
		return protocols.Response{}, err
	}

	err = networkHelper.Close()
	if err != nil {
		return protocols.Response{}, err
	}

	return response, nil
}
