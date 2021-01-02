package api

import "time"

// Representation of a query request for a specific game server.
type Request struct {
	Game    string         // The game protocol to use, can be left out for the `Detect` function.
	IP      string         // The game server's query IP
	Port    uint16         // The game server's query port
	Timeout *time.Duration // Timeout for a single send/receive operation in the game's protocol.
}

// Player information of the server
type PlayersResponse struct {
	Current int      // The amount of players currently on the server
	Max     int      // The amount of players the server can hold
	Names   []string // List of player names on the server, could be partial (so that the length of Names =/= Current)
}

// Representation of a query result for a specific game server. All of the fields are guaranteed to be present (other than the contents of Raw).
type Response struct {
	Name    string          // The server name
	Players PlayersResponse // Player information of the server

	Raw interface{} // Contains the original, raw response received from the game's protocol.
}

// Raw Minecraft UDP response
type MinecraftUDPRaw struct {
	Hostname   string
	GameType   string
	GameID     string
	Version    string
	Plugins    string
	Map        string
	NumPlayers uint16
	MaxPlayers uint16
	HostPort   uint16
	HostIP     string
	Players    []string
}

// Raw Minecraft TCP response
type MinecraftTCPRaw struct {
	Version struct {
		Name     string
		Protocol int
	}
	Players struct {
		Max    int
		Online int
		Sample []struct {
			Name string
			ID   string
		}
	}
	Description struct {
		Text string
	}
	Favicon string
}

// Optional extra data included in SourceQuery A2S info response
type SourceQuery_ExtraData struct {
	Port         uint16
	SteamID      uint64
	SourceTVPort uint16
	SourceTVName string
	Keywords     string
	GameID       uint64
}

// Raw Source Query A2S info response
type SourceQuery_A2SInfo struct {
	Protocol    uint8
	Name        string
	Map         string
	Folder      string
	Game        string
	ID          uint16
	Players     uint8
	MaxPlayers  uint8
	Bots        uint8
	ServerType  uint8
	Environment uint8
	Visibility  uint8
	VAC         uint8

	// The Ship

	Version   string
	EDF       uint8
	ExtraData SourceQuery_ExtraData
}

// Single A2S_PLAYER query's player response
type SourceQuery_A2SPlayer struct {
	Index    uint8
	Name     string
	Score    int32
	Duration float32
}
