package lcn

import (
	"encoding/hex"
	"fmt"

	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

const (
	minLcnPacketLength = 6
	lcnPacketLength8   = 8
	lcnPacketLength12  = 12
	lcnPacketLength20  = 20

	infoLengthMask  = 0x0C
	infoLengthShift = 2

	lenCode6  = 0b00
	lenCode8  = 0b01
	lenCode12 = 0b10
	lenCode20 = 0b11
)

var (
	ErrLcnPacketIncomplete      = fmt.Errorf("%w: LCN Packet to short", packet.ErrPacketIncomplete)
	ErrLcnPacketInvalid         = fmt.Errorf("%w: LCN Packet Invalid", packet.ErrPacketInvalid)
	ErrLcnPacketInvalidChecksum = fmt.Errorf("%w: LCN Checksum invalid", packet.ErrPacketInvalid)
)

var _ packet.Packet = &Packet{}

type Packet struct {
	Src      byte
	Info     byte
	Checksum byte
	Seg      byte
	Dst      byte
	Cmd      byte
	Payload  []byte
}

func getExpectedLength(info byte) int {
	switch (info & infoLengthMask) >> infoLengthShift {
	case lenCode6:
		return minLcnPacketLength
	case lenCode8:
		return lcnPacketLength8
	case lenCode12:
		return lcnPacketLength12
	case lenCode20:
		return lcnPacketLength20
	default:
		return 0
	}
}

func getLengthCode(bufLen int) (byte, bool) {
	switch bufLen {
	case minLcnPacketLength:
		return lenCode6, true
	case lcnPacketLength8:
		return lenCode8, true
	case lcnPacketLength12:
		return lenCode12, true
	case lcnPacketLength20:
		return lenCode20, true
	default:
		return 0, false
	}
}

func Deserialize(buf []byte) (packet.Packet, error) {
	if len(buf) < minLcnPacketLength {
		return nil, ErrLcnPacketIncomplete
	}

	lcn := new(Packet)

	payloadLength := len(buf) - minLcnPacketLength

	lcn.Src = mirrorSrc(buf[0])
	lcn.Info = buf[1]
	lcn.Checksum = buf[2]
	lcn.Seg = buf[3]
	lcn.Dst = buf[4]
	lcn.Cmd = buf[5]

	lcn.Payload = make([]byte, payloadLength)
	copy(lcn.Payload, buf[minLcnPacketLength:minLcnPacketLength+payloadLength])

	expectedLen := getExpectedLength(lcn.Info)

	if len(buf) < expectedLen {
		return nil, ErrLcnPacketIncomplete
	}

	if len(buf) > expectedLen {
		return nil, ErrLcnPacketInvalid
	}

	if checksum := calcChecksum(buf); checksum != lcn.Checksum {
		log.Debugf("Wrong Checksum is %x expected: %x", lcn.Checksum, checksum)

		return nil, ErrLcnPacketInvalidChecksum
	}

	log.Debugf("Deserialized LCN Packet {%s}", lcn.ToString())

	return lcn, nil
}

// this function sets checksum and length information by itself.
func (lcn *Packet) Serialize() ([]byte, error) {
	bufLen := minLcnPacketLength + len(lcn.Payload)
	buf := make([]byte, bufLen)
	buf[0] = mirrorSrc(lcn.Src)
	buf[1] = lcn.Info
	buf[2] = 0 // checksum will be set later
	buf[3] = lcn.Seg
	buf[4] = lcn.Dst
	buf[5] = lcn.Cmd
	copy(buf[minLcnPacketLength:], lcn.Payload)

	code, ok := getLengthCode(bufLen)
	if !ok {
		return nil, ErrLcnPacketInvalid
	}

	buf[1] = buf[1]&^infoLengthMask | (code << infoLengthShift)

	buf[2] = calcChecksum(buf)

	return buf, nil
}

func (lcn *Packet) ToString() string {
	return fmt.Sprintf("src: %x, info: %x, crc: %x, seg: %x, dst: %x, cmd: %x, payload: %s",
		lcn.Src, lcn.Info, lcn.Checksum, lcn.Seg, lcn.Dst, lcn.Cmd, hex.EncodeToString(lcn.Payload))
}

func (lcn *Packet) ToNiceString() string {
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

//nolint:mnd
func calcChecksum(buf []byte) byte {
	var checksum byte

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

	return checksum
}
