package protocols

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wisp-gg/gamequery/api"
	"github.com/wisp-gg/gamequery/internal"
	"sort"
)

type SourceQuery struct{}

func (sq SourceQuery) Name() string {
	return "source"
}

func (sq SourceQuery) Aliases() []string {
	return []string{}
}

func (sq SourceQuery) DefaultPort() uint16 {
	return 27015
}

func (sq SourceQuery) Priority() uint16 {
	return 1
}

func (sq SourceQuery) Network() string {
	return "udp"
}

type partialPacket struct {
	ID     int32
	Number int8
	Size   uint16
	Data   []byte
}

func (sq SourceQuery) handleMultiplePackets(helper internal.NetworkHelper, initialPacket internal.Packet) (internal.Packet, error) {
	var initial = true
	var curPacket = initialPacket
	var packets []partialPacket
	var compressed = false
	var decompressedSize, crc32 int32 = 0, 0
	for {
		if !initial {
			var err error
			curPacket, err = helper.Receive()
			if err != nil {
				return internal.Packet{}, err
			}

			curPacket.SetOrder(binary.LittleEndian)
		}

		if curPacket.ReadInt32() != -2 {
			return internal.Packet{}, errors.New("received packet isn't part of split response")
		}

		// For the sake of simplicity, we'll assume that the server is Source based instead of possibly Goldsource.
		id, total, number, size := curPacket.ReadInt32(), curPacket.ReadInt8(), curPacket.ReadInt8(), curPacket.ReadUint16()
		if initial {
			compressed = uint32(id)&0x80000000 != 0

			if compressed {
				decompressedSize, crc32 = curPacket.ReadInt32(), curPacket.ReadInt32()
			}

			initial = false
		}

		packets = append(packets, partialPacket{
			ID:     id,
			Number: number,
			Size:   size,
			Data:   curPacket.ReadRest(),
		})

		if curPacket.IsInvalid() {
			return internal.Packet{}, errors.New("split packet response was malformed")
		}

		if len(packets) == int(total) {
			break
		}
	}

	sort.Slice(packets, func(i, j int) bool {
		return packets[i].Number < packets[j].Number
	})

	packet := internal.Packet{}
	packet.SetOrder(binary.LittleEndian)
	for _, partial := range packets {
		packet.WriteRaw(partial.Data...)
	}

	if compressed {
		// TODO: Handle decompression (only engines from ~2006-era seem to implement this)

		return internal.Packet{}, errors.New("received packet that is bz2 compressed (" + string(decompressedSize) + ", " + string(crc32) + ")")
	}

	// The constructed packet will resemble the simple response format, so we need to get rid of
	// the FF FF FF FF prefix (as we'll return to logic after the initial header reading).
	packet.ReadInt32()

	return packet, nil
}

func (sq SourceQuery) handleReceivedPacket(helper internal.NetworkHelper, packet internal.Packet) (internal.Packet, error) {
	packetType := packet.ReadInt32()
	if packetType == -1 {
		return packet, nil
	}

	if packetType == -2 {
		packet.Forward(-4) // Seek back so we're able to reread the data in handleMultiplePackets

		return sq.handleMultiplePackets(helper, packet)
	}

	return internal.Packet{}, errors.New(fmt.Sprintf("unable to handle unknown packet type %d", packetType))
}

func (sq SourceQuery) request(helper internal.NetworkHelper, requestPacket internal.Packet, wantedId uint8, allowChallengeRequest bool) (internal.Packet, error) {
	if err := helper.Send(requestPacket.GetBuffer()); err != nil {
		return internal.Packet{}, err
	}

	packet, err := helper.Receive()
	if err != nil {
		return internal.Packet{}, err
	}

	packet.SetOrder(binary.LittleEndian)
	packet, err = sq.handleReceivedPacket(helper, packet)
	if err != nil {
		return internal.Packet{}, err
	}

	responseType := packet.ReadUint8()
	if responseType == wantedId {
		return packet, nil
	}

	if responseType != 0x41 {
		return internal.Packet{}, errors.New(fmt.Sprintf("unable to handle unknown response type %d", responseType))
	}

	// If a challenge response fails, the game may respond with another challenge.
	// To avoid a recursive loop, we explicitly disallow requesting new challenges after
	// a single challenge request has been done (initial request).
	if !allowChallengeRequest {
		return internal.Packet{}, errors.New("unable to handle response due to disallowing challenge requests")
	}

	challengedRequest := internal.Packet{}
	challengedRequest.SetOrder(binary.LittleEndian)
	challengedRequest.WriteInt32(requestPacket.ReadInt32())
	challengedRequest.WriteUint8(requestPacket.ReadUint8())
	challengedRequest.WriteInt32(packet.ReadInt32())

	return sq.request(helper, challengedRequest, wantedId, false)
}

func (sq SourceQuery) Execute(helper internal.NetworkHelper) (api.Response, error) {
	requestPacket := internal.Packet{}
	requestPacket.SetOrder(binary.LittleEndian)

	// A2S_INFO request
	requestPacket.WriteRaw(0xFF, 0xFF, 0xFF, 0xFF, 0x54)
	requestPacket.WriteString("Source Engine Query")
	requestPacket.WriteRaw(0x00)

	packet, err := sq.request(helper, requestPacket, 0x49, true)
	if err != nil {
		return api.Response{}, err
	}

	raw := api.SourceQuery_A2SInfo{
		Protocol:    packet.ReadUint8(),
		Name:        packet.ReadString(),
		Map:         packet.ReadString(),
		Folder:      packet.ReadString(),
		Game:        packet.ReadString(),
		ID:          packet.ReadUint16(),
		Players:     packet.ReadUint8(),
		MaxPlayers:  packet.ReadUint8(),
		Bots:        packet.ReadUint8(),
		ServerType:  packet.ReadUint8(),
		Environment: packet.ReadUint8(),
		Visibility:  packet.ReadUint8(),
		VAC:         packet.ReadUint8(),
	}

	if raw.ID == 2420 {
		return api.Response{}, errors.New("detected The Ship response, unsupported")
	}

	raw.Version = packet.ReadString()

	if !packet.ReachedEnd() {
		raw.EDF = packet.ReadUint8()

		extraData := api.SourceQuery_ExtraData{}
		if (raw.EDF & 0x80) != 0 {
			extraData.Port = packet.ReadUint16()
		}

		if (raw.EDF & 0x10) != 0 {
			extraData.SteamID = packet.ReadUint64()
		}

		if (raw.EDF & 0x40) != 0 {
			extraData.SourceTVPort = packet.ReadUint16()
			extraData.SourceTVName = packet.ReadString()
		}

		if (raw.EDF & 0x20) != 0 {
			extraData.Keywords = packet.ReadString()
		}

		if (raw.EDF & 0x01) != 0 {
			extraData.GameID = packet.ReadUint64()
		}

		raw.ExtraData = extraData
	}

	if packet.IsInvalid() {
		return api.Response{}, errors.New("received packet is invalid")
	}

	// Attempt to additionally get info from A2S_PLAYER (as it contains player names)
	// Though if this fails, just fail silently as it's acceptable for that information to be missing
	// and it's better than having no info at all.
	//
	// Depending on the game type, it may also just stop responding to A2S_PLAYER due to too many players.
	requestPacket.Clear()
	requestPacket.WriteRaw(0xFF, 0xFF, 0xFF, 0xFF, 0x55, 0xFF, 0xFF, 0xFF, 0xFF)

	packet, err = sq.request(helper, requestPacket, 0x44, true)
	var playerList []string
	if err == nil {
		packet.ReadUint8() // Number of players we received information for

		for {
			player := api.SourceQuery_A2SPlayer{
				Index:    packet.ReadUint8(),
				Name:     packet.ReadString(),
				Score:    packet.ReadInt32(),
				Duration: packet.ReadFloat32(),
			}

			if packet.IsInvalid() {
				break
			}

			playerList = append(playerList, player.Name)

			if packet.ReachedEnd() {
				break
			}
		}
	}

	return api.Response{
		Name: raw.Name,
		Players: api.PlayersResponse{
			Current: int(raw.Players),
			Max:     int(raw.MaxPlayers),
			Names:   playerList,
		},

		Raw: raw,
	}, nil
}
