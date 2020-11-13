package gamequery

import (
	"errors"
	"github.com/wisp-gg/gamequery/protocols"
	"sort"
	"sync"
	"time"
)

// Representation of a query request for a specific game server.
type Request struct {
	Game    string         // The game protocol to use, can be left out for the `Detect` function.
	IP      string         // The game server's query IP
	Port    uint16         // The game server's query port
	Timeout *time.Duration // Timeout for a single send/receive operation in the game's protocol.
}

var queryProtocols = []protocols.Protocol{
	protocols.SourceQuery{},
	protocols.MinecraftUDP{},
	protocols.MinecraftTCP{},
}

func findProtocols(name string) []protocols.Protocol {
	found := make([]protocols.Protocol, 0)
	for _, protocol := range queryProtocols {
		if protocol.Name() == name {
			found = append(found, protocol)
		} else {
			for _, protocolName := range protocol.Aliases() {
				if protocolName == name {
					found = append(found, protocol)
				}
			}
		}
	}

	return found
}

type queryResult struct {
	Name     string
	Priority uint16
	Err      error
	Response protocols.Response
}

// Query the game server by detecting the protocol (trying all available protocols).
// This usually should be used as the initial query function and then use `Query` function
// with the returned protocol if the query succeeds. Otherwise each function call will take always
// <req.Timeout> duration even if the response was received earlier from one of the protocols.
func Detect(req Request) (protocols.Response, string, error) {
	return query(req, queryProtocols)
}

// Query the game server using the protocol provided in req.Game.
func Query(req Request) (protocols.Response, error) {
	chosenProtocols := findProtocols(req.Game)
	if len(chosenProtocols) < 1 {
		return protocols.Response{}, errors.New("could not find protocols for the game")
	}

	response, _, err := query(req, chosenProtocols)
	return response, err
}

func query(req Request, chosenProtocols []protocols.Protocol) (protocols.Response, string, error) {
	var wg sync.WaitGroup
	wg.Add(len(chosenProtocols))

	queryResults := make([]queryResult, len(chosenProtocols))
	for index, queryProtocol := range chosenProtocols {
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
				Name:     queryProtocol.Name(),
				Priority: queryProtocol.Priority(),
				Err:      nil,
				Response: response,
			}
		}(queryProtocol, index)
	}

	wg.Wait()
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
			return result.Response, result.Name, nil
		}
	}

	return protocols.Response{}, "", firstError
}
