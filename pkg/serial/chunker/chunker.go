package chunker

import (
	"bytes"
	"errors"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
)

var (
	ErrIncompleteLcn = errors.New("incomplete LCN packet")
	ErrInvalidLcn    = errors.New("invalid LCN packet")
	ErrInvalidLcnCRC = errors.New("invalid CRC on LCN packet")
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
		c.buffer.Reset()
	}

	search := func() {
		c.buffer.Next(1)
	}

read_loop:
	for _, b := range buf {
		c.buffer.WriteByte(b)

		for c.buffer.Len() >= c.minLength {
			pkt, err := c.deserializer(c.buffer.Bytes())
			if err != nil {
				switch {
				case errors.Is(err, packet.ErrPacketIncomplete):
					continue read_loop
				default:
					log.Errorf("%s 0x%x", err, c.buffer.Bytes())
					search()

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
