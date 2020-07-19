package gamequery

import (
	"errors"
	"github.com/wisp-gg/gamequery/protocols"
	"sort"
	"sync"
	"time"
)

type Request struct {
	Game    string
	IP      string
	Port    uint16
	Timeout *time.Duration
}

var queryProtocols = []protocols.Protocol{
	protocols.SourceQuery{},
	protocols.MinecraftUDP{},
	protocols.MinecraftTCP{},
}

func findProtocols(name string) []protocols.Protocol {
	found := make([]protocols.Protocol, 0)
	for _, protocol := range queryProtocols {
		for _, protocolName := range protocol.Names() {
			if protocolName == name {
				found = append(found, protocol)
			}
		}
	}

	return found
}

type queryResult struct {
	Priority uint16
	Err      error
	Response protocols.Response
}

func Query(req Request) (protocols.Response, error) {
	queryProtocols := findProtocols(req.Game)
	if len(queryProtocols) < 1 {
		return protocols.Response{}, errors.New("could not find protocols for the game")
	}

	var wg sync.WaitGroup
	wg.Add(len(queryProtocols))

	queryResults := make([]queryResult, len(queryProtocols))
	for index, queryProtocol := range queryProtocols {
		go func(queryProtocol protocols.Protocol, index int) {
			defer wg.Done()

			var port = queryProtocol.DefaultPort()
			if req.Port != 0 {
				port = req.Port
			}

			var timeout = 5 * time.Second
			if req.Timeout != nil {
				timeout = *req.Timeout
			}

			networkHelper := protocols.NetworkHelper{}
			if err := networkHelper.Initialize(queryProtocol.Network(), req.IP, port, timeout); err != nil {
				queryResults[index] = queryResult{
					Priority: queryProtocol.Priority(),
					Err:      err,
					Response: protocols.Response{},
				}
				return
			}
			defer networkHelper.Close()

			response, err := queryProtocol.Execute(networkHelper)
			if err != nil {
				queryResults[index] = queryResult{
					Priority: queryProtocol.Priority(),
					Err:      err,
					Response: protocols.Response{},
				}
				return
			}

			queryResults[index] = queryResult{
				Priority: queryProtocol.Priority(),
				Err:      nil,
				Response: response,
			}
		}(queryProtocol, index)
	}

	wg.Wait() // TODO: Somehow skip waiting for other protocols if we have a response?
	sort.Slice(queryResults, func(i, j int) bool {
		return queryResults[i].Priority > queryResults[j].Priority
	})

	var firstError error
	for _, result := range queryResults {
		if result.Err != nil {
			if firstError == nil {
				firstError = result.Err
			}
		} else {
			return result.Response, nil
		}
	}

	return protocols.Response{}, firstError
}
