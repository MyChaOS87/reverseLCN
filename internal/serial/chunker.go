package serial

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
)

var (
	ErrIncompleteLcn = errors.New("Incomplete LCN Packet")
	ErrInvalidLcn    = errors.New("Invalid LCN Packet")
	ErrInvalidLcnCRC = errors.New("Invalid CRC on LCN Packet")
)

const (
	MIN_LCN_PACKET_LENGTH = 6
	MAX_LCN_PACKET_LENGTH = 20
)

type EjectFunc func(LcnPacket)

type LcnPacket struct {
	Src     byte
	Info    byte
	crc     byte
	Seg     byte
	Dst     byte
	Cmd     byte
	Payload []byte
}

type chunker struct {
	buffer bytes.Buffer
	start  time.Time
}

func (c *chunker) collect(buf []byte, eject EjectFunc) {
	// TODO: timeout perhaps no more neccessary if we learn proper parsing of packets

	// now := time.Now()
	// if c.buffer.Len() == 0 {
	// 	c.start = now
	// }

	// timeout := c.start.Add(16 * time.Millisecond)
	reset := func() {
		c.buffer.Truncate(0)
	}

	for _, b := range buf {
		c.buffer.WriteByte(b)

		if c.buffer.Len() > MIN_LCN_PACKET_LENGTH {
			pkt, err := newLcnPacketFromBuffer(c.buffer.Bytes())
			if err != nil {
				switch err {
				case ErrIncompleteLcn:
					continue
				default:
					log.Errorf("%s %x", err, c.buffer.Bytes())
					reset()
					continue
				}
			}

			eject(*pkt)
			reset()
		}
		// if now.After(timeout) {
		// 	eject(c.buffer.String())

		// 	c.buffer.Truncate(0)
		// 	c.start = now
		// 	timeout = c.start.Add(16 * time.Millisecond)
		// }
	}
}

func newLcnPacketFromBuffer(buf []byte) (*LcnPacket, error) {
	if len(buf) < MIN_LCN_PACKET_LENGTH {
		return nil, ErrIncompleteLcn
	}

	mirrorSrc := func(in byte) byte {
		src := byte(0)
		for p := 0; p < 8; p++ {
			src <<= 1
			src += (in & (1 << p) >> p)
		}
		return src
	}

	lcn := new(LcnPacket)

	payloadLength := len(buf) - MIN_LCN_PACKET_LENGTH

	lcn.Payload = make([]byte, payloadLength)
	lcn.Src = mirrorSrc(buf[0])
	lcn.Info = buf[1]
	lcn.crc = buf[2]
	lcn.Seg = buf[3]
	lcn.Dst = buf[4]
	lcn.Cmd = buf[5]

	copy(lcn.Payload, buf[MIN_LCN_PACKET_LENGTH:MIN_LCN_PACKET_LENGTH+payloadLength])

	expectedLen := 8
	// just a guess
	switch lcn.Info & 0xc >> 2 {
	case 0:
		expectedLen = 6
	case 1:
		expectedLen = 8
	case 2:
		expectedLen = 12
	case 3:
		expectedLen = 20
	}

	if len(buf) < expectedLen {
		return nil, ErrIncompleteLcn
	}

	if len(buf) > expectedLen {
		return nil, ErrInvalidLcn
	}

	log.Debugf("lenght: %d info: %x", lcn.Info&0xc>>2, lcn.Info)

	calcChecksum := func(buf []byte) int {
		var checksum byte = 0

		for i, b := range buf {
			if i == 2 {
				continue
			}

			tmp := int(b) + int(checksum)
			tmp2 := ((tmp&0x7f)<<2 | (tmp&0x180)>>7)
			if tmp2 > 0xff {
				tmp2 -= 0xff
			}
			checksum = byte(tmp2)
		}

		return int(checksum)
	}

	checksum := calcChecksum(buf)
	if checksum != int(lcn.crc) {
		log.Debugf("Wrong Checksum is %x expected: %x", buf[2], checksum)

		return nil, ErrInvalidLcnCRC
	}

	return lcn, nil
}

func (lcn *LcnPacket) ToString() string {
	return fmt.Sprintf("got: src: %x, info: %x, crc: %x, seg: %x, dst: %x, cmd: %x, payload: %s",
		lcn.Src, lcn.Info, lcn.crc, lcn.Seg, lcn.Dst, lcn.Cmd, hex.EncodeToString(lcn.Payload))
}

func (lcn *LcnPacket) ToNiceString() string {
	return fmt.Sprintf("%2x->%2x:%2x cmd: %2x, payload: %s",
		lcn.Src, lcn.Seg, lcn.Dst, lcn.Cmd, hex.EncodeToString(lcn.Payload))
}
