package protocols

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/wisp-gg/gamequery/api"
	"github.com/wisp-gg/gamequery/internal"
	"math/rand"
	"strconv"
	"time"
)

type MinecraftUDP struct{}

func (mc MinecraftUDP) Name() string {
	return "minecraft_udp"
}

func (mc MinecraftUDP) Aliases() []string {
	return []string{
		"minecraft",
	}
}

func (mc MinecraftUDP) DefaultPort() uint16 {
	return 25565
}

func (mc MinecraftUDP) Priority() uint16 {
	return 10
}

func (mc MinecraftUDP) Network() string {
	return "udp"
}

func generateSessionID() int32 {
	rand.Seed(time.Now().UTC().UnixNano())

	return rand.Int31() & 0x0F0F0F0F
}

func parseChallengeToken(challengeToken string) ([]byte, error) {
	parsedInt, err := strconv.ParseInt(challengeToken, 10, 32)
	if err != nil {
		return []byte{}, err
	}

	buf := &bytes.Buffer{}
	err = binary.Write(buf, binary.BigEndian, parsedInt)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes()[buf.Len()-4:], nil
}

func (mc MinecraftUDP) Execute(helper internal.NetworkHelper) (api.Response, error) {
	sessionId := generateSessionID()

	packet := internal.Packet{}
	packet.SetOrder(binary.BigEndian)
	packet.WriteRaw(0xFE, 0xFD, 0x09)
	packet.WriteInt32(sessionId)

	err := helper.Send(packet.GetBuffer())
	if err != nil {
		return api.Response{}, err
	}

	handshakePacket, err := helper.Receive()
	if err != nil {
		return api.Response{}, err
	}

	handshakePacket.SetOrder(binary.BigEndian)
	if handshakePacket.ReadUint8() != 0x09 {
		return api.Response{}, errors.New("sent a handshake, but didn't receive handshake response back")
	}

	if handshakePacket.ReadInt32() != sessionId {
		return api.Response{}, errors.New("received handshake for wrong session id")
	}

	challengeToken, err := parseChallengeToken(handshakePacket.ReadString())
	if err != nil {
		return api.Response{}, err
	}

	packet.Clear()
	packet.WriteRaw(0xFE, 0xFD, 0x00)
	packet.WriteInt32(sessionId)
	packet.WriteRaw(challengeToken...)
	packet.WriteRaw(0x00, 0x00, 0x00, 0x00)

	err = helper.Send(packet.GetBuffer())
	if err != nil {
		return api.Response{}, err
	}

	responsePacket, err := helper.Receive()
	if err != nil {
		return api.Response{}, err
	}

	responsePacket.SetOrder(binary.BigEndian)
	if responsePacket.ReadUint8() != 0x00 {
		return api.Response{}, errors.New("sent a full stat request, but didn't receive stat response back")
	}

	if responsePacket.ReadInt32() != sessionId {
		return api.Response{}, errors.New("received handshake for wrong session id")
	}

	responsePacket.Forward(11)

	raw := api.MinecraftUDPRaw{}
	for {
		key := responsePacket.ReadString()
		if key == "" {
			break
		}

		val := responsePacket.ReadString()

		switch key {
		case "hostname":
			raw.Hostname = val
		case "gametype":
			raw.GameType = val
		case "game_id":
			raw.GameID = val
		case "version":
			raw.Version = val
		case "plugins":
			raw.Plugins = val
		case "map":
			raw.Map = val
		case "numplayers":
			tmp, _ := strconv.ParseInt(val, 10, 16)
			raw.NumPlayers = uint16(tmp)
		case "maxplayers":
			tmp, _ := strconv.ParseInt(val, 10, 16)
			raw.MaxPlayers = uint16(tmp)
		case "hostport":
			tmp, _ := strconv.ParseInt(val, 10, 16)
			raw.HostPort = uint16(tmp)
		case "hostip":
			raw.HostIP = val
		}
	}

	responsePacket.Forward(10)

	for {
		playerName := responsePacket.ReadString()
		if playerName == "" {
			break
		}

		raw.Players = append(raw.Players, playerName)
	}

	if responsePacket.IsInvalid() {
		return api.Response{}, errors.New("received packet is invalid")
	}

	return api.Response{
		Name: raw.Hostname,
		Players: api.PlayersResponse{
			Current: int(raw.NumPlayers),
			Max:     int(raw.MaxPlayers),
			Names:   raw.Players,
		},

		Raw: raw,
	}, nil
}
