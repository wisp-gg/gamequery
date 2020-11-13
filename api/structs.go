package api

import "time"

// Representation of a query request for a specific game server.
type Request struct {
	Game    string         // The game protocol to use, can be left out for the `Detect` function.
	IP      string         // The game server's query IP
	Port    uint16         // The game server's query port
	Timeout *time.Duration // Timeout for a single send/receive operation in the game's protocol.
}

// Representation of a query result for a specific game server.
type Response struct {
	// TODO: Decide the structure of this

	Raw interface{} // Contains the original, raw response received from the game's protocol.
}
