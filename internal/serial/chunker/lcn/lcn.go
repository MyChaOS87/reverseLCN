package lcn

import (
	"encoding/hex"
	"fmt"

	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

const (
	// MinPacketLength is the wire length in bytes of the smallest LCN packet (header only, empty payload).
	MinPacketLength   = 6
	lcnPacketLength8  = 8
	lcnPacketLength12 = 12
	lcnPacketLength20 = 20

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
		return MinPacketLength
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
	case MinPacketLength:
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

func Deserialize(buf []byte) (packet.Packet, int, error) {
	if len(buf) < MinPacketLength {
		return nil, 0, ErrLcnPacketIncomplete
	}

	expectedLen := getExpectedLength(buf[1])
	if expectedLen < MinPacketLength {
		return nil, 0, ErrLcnPacketInvalid
	}

	if len(buf) < expectedLen {
		return nil, 0, ErrLcnPacketIncomplete
	}

	frame := buf[:expectedLen]

	if checksum := calcChecksum(frame); checksum != frame[2] {
		log.Debugf("Wrong Checksum is %x expected: %x", frame[2], checksum)

		return nil, 0, ErrLcnPacketInvalidChecksum
	}

	payloadLength := expectedLen - MinPacketLength
	payload := make([]byte, payloadLength)
	copy(payload, frame[MinPacketLength:expectedLen])

	lcn := &Packet{
		Src:      mirrorSrc(frame[0]),
		Info:     frame[1],
		Checksum: frame[2],
		Seg:      frame[3],
		Dst:      frame[4],
		Cmd:      frame[5],
		Payload:  payload,
	}

	log.Debugf("Deserialized LCN Packet {%s}", lcn.ToString())

	return lcn, expectedLen, nil
}

// Serialize encodes the packet and sets checksum and length information by itself.
func (lcn *Packet) Serialize() ([]byte, error) {
	bufLen := MinPacketLength + len(lcn.Payload)
	buf := make([]byte, bufLen)
	buf[0] = mirrorSrc(lcn.Src)
	buf[1] = lcn.Info
	buf[2] = 0 // checksum will be set later
	buf[3] = lcn.Seg
	buf[4] = lcn.Dst
	buf[5] = lcn.Cmd
	copy(buf[MinPacketLength:], lcn.Payload)

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

	for p := range 8 {
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

		checksum = byte(tmp2 & 0xFF)
	}

	return checksum
}
