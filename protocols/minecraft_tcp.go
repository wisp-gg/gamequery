package protocols

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

type MinecraftTCP struct{}

type MinecraftTCPVersion struct {
	Name     string
	Protocol int
}

type MinecraftTCPPlayer struct {
	Name string
	ID   string
}

type MinecraftTCPPlayers struct {
	Max    int
	Online int
	Sample []MinecraftTCPPlayer
}

type MinecraftTCPDescription struct {
	Text string
}

type MinecraftTCPRaw struct {
	Version     MinecraftTCPVersion
	Players     MinecraftTCPPlayers
	Description MinecraftTCPDescription
	Favicon     string
}

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

func buildMCPacket(bulkData ...interface{}) *Packet {
	packet := Packet{}
	packet.SetOrder(binary.BigEndian)

	tmpPacket := Packet{}
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
			fmt.Printf("unhandled %s\n", val)
		}
	}

	packet.WriteVarint(tmpPacket.Length())
	packet.WriteRaw(tmpPacket.GetBuffer()...)

	return &packet
}

func (mc MinecraftTCP) Execute(helper NetworkHelper) (Response, error) {
	err := helper.Send(buildMCPacket([]byte{0x00, 0x00}, helper.GetIP(), helper.GetPort(), 0x01).GetBuffer())
	if err != nil {
		return Response{}, err
	}

	err = helper.Send(buildMCPacket(0x00).GetBuffer())
	if err != nil {
		return Response{}, err
	}

	responsePacket, err := helper.Receive()
	if err != nil {
		return Response{}, err
	}

	packetLength := responsePacket.ReadVarint()
	packetId := responsePacket.ReadVarint()
	if packetId != 0 {
		return Response{}, errors.New("received something else than a status response")
	}

	if packetId > packetLength {
		responsePacket.ReadVarint() // No idea what this is
	}

	responsePacket.ReadVarint() // Actual JSON strings' length (unneeded with ReadString)

	jsonBody := responsePacket.ReadString()

	raw := MinecraftTCPRaw{}
	err = json.Unmarshal([]byte(jsonBody), &raw)
	if err != nil {
		return Response{}, err
	}

	if responsePacket.IsInvalid() {
		return Response{}, errors.New("received packet is invalid")
	}

	return Response{
		Raw: raw,
	}, nil
}
