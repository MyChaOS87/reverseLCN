package lcn

import (
	"encoding/hex"
	"fmt"

	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/packet"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/pkg/errors"
)

const (
	MIN_LCN_PACKET_LENGTH = 6
)

var (
	ErrLcnPacketIncomplete      = errors.Wrap(packet.ErrPacketIncomplete, "LCN Packet to short")
	ErrLcnPacketInvalid         = errors.Wrap(packet.ErrPacketInvalid, "LCN Packet Invalid")
	ErrLcnPacketInvalidChecksum = errors.Wrap(packet.ErrPacketInvalid, "LCN Checksum invalid")
)

var _ packet.Packet = &LcnPacket{}

var lengthMapping = map[byte]int{
	0b00: 6,
	0b01: 8,
	0b10: 12,
	0b11: 20,
}

type LcnPacket struct {
	Src      byte
	Info     byte
	Checksum byte
	Seg      byte
	Dst      byte
	Cmd      byte
	Payload  []byte
}

func Deserialize(buf []byte) (packet.Packet, error) {
	if len(buf) < MIN_LCN_PACKET_LENGTH {
		return nil, ErrLcnPacketIncomplete
	}

	lcn := new(LcnPacket)

	payloadLength := len(buf) - MIN_LCN_PACKET_LENGTH

	lcn.Src = mirrorSrc(buf[0])
	lcn.Info = buf[1]
	lcn.Checksum = buf[2]
	lcn.Seg = buf[3]
	lcn.Dst = buf[4]
	lcn.Cmd = buf[5]

	lcn.Payload = make([]byte, payloadLength)
	copy(lcn.Payload, buf[MIN_LCN_PACKET_LENGTH:MIN_LCN_PACKET_LENGTH+payloadLength])

	expectedLen := lengthMapping[lcn.Info&0xC>>2]

	if len(buf) < expectedLen {
		return nil, ErrLcnPacketIncomplete
	}

	if len(buf) > expectedLen {
		return nil, ErrLcnPacketInvalid
	}

	if checksum := calcChecksum(buf); checksum != int(lcn.Checksum) {
		log.Debugf("Wrong Checksum is %x expected: %x", lcn.Checksum, checksum)

		return nil, ErrLcnPacketInvalidChecksum
	}

	log.Debugf("Deserialized LCN Packet {%s}", lcn.ToString())
	return lcn, nil
}

func (lcn *LcnPacket) Serialize() []byte {
	panic("not implemented")
}

func (lcn *LcnPacket) ToString() string {
	return fmt.Sprintf("src: %x, info: %x, crc: %x, seg: %x, dst: %x, cmd: %x, payload: %s",
		lcn.Src, lcn.Info, lcn.Checksum, lcn.Seg, lcn.Dst, lcn.Cmd, hex.EncodeToString(lcn.Payload))
}

func (lcn *LcnPacket) ToNiceString() string {
	return fmt.Sprintf("%2x->%2x:%2x cmd: %2x, payload: %s",
		lcn.Src, lcn.Seg, lcn.Dst, lcn.Cmd, hex.EncodeToString(lcn.Payload))
}

func mirrorSrc(in byte) byte {
	src := byte(0)
	for p := 0; p < 8; p++ {
		src <<= 1
		src += (in & (1 << p) >> p)
	}
	return src
}

func calcChecksum(buf []byte) int {
	var checksum byte = 0

	for i, b := range buf {
		if i == 2 {
			continue
		}

		tmp := int(b) + int(checksum)
		tmp2 := ((tmp&0x7F)<<2 | (tmp&0x180)>>7)
		if tmp2 > 0xFF {
			tmp2 -= 0xFF
		}
		checksum = byte(tmp2)
	}

	return int(checksum)
}
