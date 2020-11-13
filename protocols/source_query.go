package protocols

import (
	"encoding/binary"
	"errors"
	"sort"
)

type SourceQuery struct{}

type ExtraData struct {
	Port         uint16
	SteamID      uint64
	SourceTVPort uint16
	SourceTVName string
	Keywords     string
	GameID       uint64
}

type A2SInfo struct {
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
	ExtraData ExtraData
}

func (sq SourceQuery) Names() []string {
	return []string{
		"source",
	}
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

func (sq SourceQuery) handleMultiplePackets(helper NetworkHelper, initialPacket Packet) (Packet, error) {
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
				return Packet{}, err
			}

			curPacket.SetOrder(binary.LittleEndian)
		}

		if curPacket.ReadInt32() != -2 {
			return Packet{}, errors.New("received packet isn't part of split response")
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
			return Packet{}, errors.New("split packet response was malformed")
		}

		if len(packets) == int(total) {
			break
		}
	}

	sort.Slice(packets, func(i, j int) bool {
		return packets[i].Number < packets[j].Number
	})

	packet := Packet{}
	packet.SetOrder(binary.LittleEndian)
	for _, partial := range packets {
		packet.WriteRaw(partial.Data...)
	}

	if compressed {
		// TODO: Handle decompression (only engines from ~2006-era seem to implement this)

		return Packet{}, errors.New("received packet that is bz2 compressed (" + string(decompressedSize) + ", " + string(crc32) + ")")
	}

	// The constructed packet will resemble the simple response format, so we need to get rid of
	// the FF FF FF FF prefix (as we'll return to logic after the initial header reading).
	packet.ReadInt32()

	return packet, nil
}

func (sq SourceQuery) Execute(helper NetworkHelper) (Response, error) {
	packet := Packet{}
	packet.WriteRaw(0xFF, 0xFF, 0xFF, 0xFF, 0x54)
	packet.WriteString("Source Engine Query")
	packet.WriteRaw(0x00)

	if err := helper.Send(packet.GetBuffer()); err != nil {
		return Response{}, err
	}

	packet, err := helper.Receive()
	if err != nil {
		return Response{}, err
	}

	packet.SetOrder(binary.LittleEndian)

	if packet.ReadInt32() != -1 {
		packet.Forward(-4) // Seek back so we're able to reread the data in handleMultiplePackets

		packet, err = sq.handleMultiplePackets(helper, packet)
		if err != nil {
			return Response{}, err
		}
	}

	if packet.ReadUint8() != 0x49 {
		return Response{}, errors.New("received packet isn't a response to A2S_INFO")
	}

	raw := A2SInfo{
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
		return Response{}, errors.New("detected The Ship response, unsupported")
	}

	raw.Version = packet.ReadString()

	if !packet.ReachedEnd() {
		raw.EDF = packet.ReadUint8()

		extraData := ExtraData{}
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
		return Response{}, errors.New("received packet is invalid")
	}

	return Response{
		Raw: raw,
	}, nil
}
