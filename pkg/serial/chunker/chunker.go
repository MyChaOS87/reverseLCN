package chunker

import (
	"bytes"
	"errors"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
)

var (
	ErrIncompleteLcn = errors.New("Incomplete LCN Packet")
	ErrInvalidLcn    = errors.New("Invalid LCN Packet")
	ErrInvalidLcnCRC = errors.New("Invalid CRC on LCN Packet")
)

type EjectFunc func(packet.Packet)

type Chunker interface {
	Collect(buf []byte, eject EjectFunc)
}

type chunker struct {
	deserializer packet.Deserializer
	minLength    int

	buffer bytes.Buffer
}

func (c *chunker) Collect(buf []byte, eject EjectFunc) {
	reset := func() {
		c.buffer.Truncate(0)
	}

	for _, b := range buf {
		c.buffer.WriteByte(b)

		if c.buffer.Len() >= c.minLength {
			pkt, err := c.deserializer(c.buffer.Bytes())
			if err != nil {
				switch {
				case errors.Is(err, packet.ErrPacketIncomplete):
					continue
				default:
					log.Errorf("%s 0x%x", err, c.buffer.Bytes())
					reset()
					continue
				}
			}

			eject(pkt)
			reset()
		}
	}
}

func NewChunker(deserializer packet.Deserializer, minLength int) Chunker {
	return &chunker{
		deserializer: deserializer,
		minLength:    minLength,
	}
}
