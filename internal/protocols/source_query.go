package protocols

import (
	"encoding/binary"
	"errors"
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

func (sq SourceQuery) Execute(helper internal.NetworkHelper) (api.Response, error) {
	packet := internal.Packet{}
	packet.WriteRaw(0xFF, 0xFF, 0xFF, 0xFF, 0x54)
	packet.WriteString("Source Engine Query")
	packet.WriteRaw(0x00)

	if err := helper.Send(packet.GetBuffer()); err != nil {
		return api.Response{}, err
	}

	packet, err := helper.Receive()
	if err != nil {
		return api.Response{}, err
	}

	packet.SetOrder(binary.LittleEndian)

	if packet.ReadInt32() != -1 {
		packet.Forward(-4) // Seek back so we're able to reread the data in handleMultiplePackets

		packet, err = sq.handleMultiplePackets(helper, packet)
		if err != nil {
			return api.Response{}, err
		}
	}

	if packet.ReadUint8() != 0x49 {
		return api.Response{}, errors.New("received packet isn't a response to A2S_INFO")
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

	return api.Response{
		Raw: raw,
	}, nil
}
