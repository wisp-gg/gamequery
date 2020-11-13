# gamequery
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