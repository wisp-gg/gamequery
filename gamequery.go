package gamequery

import (
	"errors"
	"github.com/wisp-gg/gamequery/api"
	"github.com/wisp-gg/gamequery/internal"
	"github.com/wisp-gg/gamequery/internal/protocols"
	"sort"
	"sync"
	"time"
)

var queryProtocols = []internal.Protocol{
	protocols.SourceQuery{},
	protocols.MinecraftUDP{},
	protocols.MinecraftTCP{},
}

func findProtocols(name string) []internal.Protocol {
	found := make([]internal.Protocol, 0)
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
	Response api.Response
}

// Query the game server by detecting the protocol (trying all available protocols).
// This usually should be used as the initial query function and then use `Query` function
// with the returned protocol if the query succeeds. Otherwise each function call will take always
// <req.Timeout> duration even if the response was received earlier from one of the protocols.
func Detect(req api.Request) (api.Response, string, error) {
	return query(req, queryProtocols)
}

// Query the game server using the protocol provided in req.Game.
func Query(req api.Request) (api.Response, error) {
	chosenProtocols := findProtocols(req.Game)
	if len(chosenProtocols) < 1 {
		return api.Response{}, errors.New("could not find protocols for the game")
	}

	response, _, err := query(req, chosenProtocols)
	return response, err
}

func query(req api.Request, chosenProtocols []internal.Protocol) (api.Response, string, error) {
	var wg sync.WaitGroup
	wg.Add(len(chosenProtocols))

	queryResults := make([]queryResult, len(chosenProtocols))
	for index, queryProtocol := range chosenProtocols {
		go func(queryProtocol internal.Protocol, index int) {
			defer wg.Done()

			var port = queryProtocol.DefaultPort()
			if req.Port != 0 {
				port = req.Port
			}

			var timeout = 5 * time.Second
			if req.Timeout != nil {
				timeout = *req.Timeout
			}

			networkHelper := internal.NetworkHelper{}
			if err := networkHelper.Initialize(queryProtocol.Network(), req.IP, port, timeout); err != nil {
				queryResults[index] = queryResult{
					Priority: queryProtocol.Priority(),
					Err:      err,
					Response: api.Response{},
				}
				return
			}
			defer networkHelper.Close()

			response, err := queryProtocol.Execute(networkHelper)
			if err != nil {
				queryResults[index] = queryResult{
					Priority: queryProtocol.Priority(),
					Err:      err,
					Response: api.Response{},
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

	return api.Response{}, "", firstError
}
