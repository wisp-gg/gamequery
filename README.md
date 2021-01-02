# gamequery [![GoDoc](https://godoc.org/wisp-gg/gamequery?status.svg)](https://godoc.org/github.com/wisp-gg/gamequery)
A Golang package for querying game servers  

## Supported protocols:
Source Query  
Minecraft TCP & UDP  

## Sample code:
```go
package main

import (
	"fmt"
	"github.com/wisp-gg/gamequery"
	"github.com/wisp-gg/gamequery/api"
)

func main() {
	res, protocol, err := gamequery.Detect(api.Request{
		IP: "127.0.0.1",
		Port: 27015,
	})
	if err != nil {
		fmt.Printf("failed to query: %s", err)
		return
	}

	fmt.Printf("Detected the protocol: %s\n", protocol)
	fmt.Printf("%+v\n", res)
}
```

NOTE: Ideally, you'd only want to use `gamequery.Detect` only once (or until one successful response), and then use `gamequery.Query` with the protocol provided.
Otherwise, each `gamequery.Detect` call will try to query the game server with _all_ possible protocols.