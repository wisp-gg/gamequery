package protocols

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wisp-gg/gamequery/api"
	"github.com/wisp-gg/gamequery/internal"
)

type MinecraftTCP struct{}

func (mc MinecraftTCP) Name() string {
	return "minecraft_tcp"
}

func (mc MinecraftTCP) Aliases() []string {
	return []string{
		"minecraft",
	}
}

func (mc MinecraftTCP) DefaultPort() uint16 {
	return 25565
}

func (mc MinecraftTCP) Priority() uint16 {
	return 1
}

func (mc MinecraftTCP) Network() string {
	return "tcp"
}

func buildMCPacket(bulkData ...interface{}) *internal.Packet {
	packet := internal.Packet{}
	packet.SetOrder(binary.BigEndian)

	tmpPacket := internal.Packet{}
	tmpPacket.SetOrder(binary.BigEndian)
	for _, data := range bulkData {
		switch val := data.(type) {
		case string:
			tmpPacket.WriteVarint(len(val))
			tmpPacket.WriteString(val)
		case int:
			tmpPacket.WriteVarint(val)
		case uint16:
			tmpPacket.WriteUint16(val)
		case []byte:
			tmpPacket.WriteRaw(val...)
		default:
			fmt.Printf("gamequery: unhandled type %s for Minecraft TCP packet, ignoring...\n", val)
		}
	}

	packet.WriteVarint(tmpPacket.Length())
	packet.WriteRaw(tmpPacket.GetBuffer()...)

	return &packet
}

func (mc MinecraftTCP) Execute(helper internal.NetworkHelper) (api.Response, error) {
	err := helper.Send(buildMCPacket([]byte{0x00, 0x00}, helper.GetIP(), helper.GetPort(), 0x01).GetBuffer())
	if err != nil {
		return api.Response{}, err
	}

	err = helper.Send(buildMCPacket(0x00).GetBuffer())
	if err != nil {
		return api.Response{}, err
	}

	responsePacket, err := helper.Receive()
	if err != nil {
		return api.Response{}, err
	}

	packetLength := responsePacket.ReadVarint()
	packetId := responsePacket.ReadVarint()
	if packetId != 0 {
		return api.Response{}, errors.New("received something else than a status response")
	}

	if packetId > packetLength {
		responsePacket.ReadVarint() // No idea what this is
	}

	responsePacket.ReadVarint() // Actual JSON strings' length (unneeded with ReadString)
	jsonBody := responsePacket.ReadString()

	if responsePacket.IsInvalid() {
		return api.Response{}, errors.New("received packet is invalid")
	}

	raw := api.MinecraftTCPRaw{}
	err = json.Unmarshal([]byte(jsonBody), &raw)
	if err != nil {
		return api.Response{}, err
	}

	var playerList []string
	for _, player := range raw.Players.Sample {
		playerList = append(playerList, player.Name)
	}

	return api.Response{
		Name: raw.Version.Name,
		Players: api.PlayersResponse{
			Current: raw.Players.Online,
			Max:     raw.Players.Max,
			Names:   playerList,
		},

		Raw: raw,
	}, nil
}
