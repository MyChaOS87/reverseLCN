package chunker

import (
	"bytes"
	"errors"

	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

type EjectFunc func(packet.Packet)

type Chunker interface {
	Collect(buf []byte, eject EjectFunc)
	Flush(eject EjectFunc)
}

type chunker struct {
	deserializer packet.Deserializer
	minLength    int

	buffer bytes.Buffer
}

func (c *chunker) Collect(buf []byte, eject EjectFunc) {
read_loop:
	for _, b := range buf {
		c.buffer.WriteByte(b)

		for c.buffer.Len() >= c.minLength && c.buffer.Len() > 0 {
			pkt, consumed, err := c.deserializer(c.buffer.Bytes())
			if err != nil {
				switch {
				case errors.Is(err, packet.ErrPacketIncomplete):
					continue read_loop
				default:
					log.Errorf("%s 0x%x", err, c.buffer.Bytes())
					c.buffer.Next(1)

					continue
				}
			}

			if consumed <= 0 {
				log.Errorf("deserializer consumed %d bytes on 0x%x", consumed, c.buffer.Bytes())
				c.buffer.Next(1)

				continue
			}

			eject(pkt)
			c.buffer.Next(consumed)
		}
	}
}

// Flush drains stalled bytes when the serial line goes idle, recovering shorter valid frames
// that follow leading noise bytes and discarding any trailing truncated fragment.
func (c *chunker) Flush(eject EjectFunc) {
	if c.buffer.Len() == 0 {
		return
	}

	for c.buffer.Len() >= c.minLength && c.buffer.Len() > 0 {
		pkt, consumed, err := c.deserializer(c.buffer.Bytes())
		if err != nil {
			log.Warnf("Discarding incomplete or invalid serial byte (buffer: 0x%x): %s", c.buffer.Bytes(), err)
			c.buffer.Next(1)

			continue
		}

		if consumed <= 0 {
			log.Warnf("Deserialized zero-length packet during flush (buffer: 0x%x)", c.buffer.Bytes())
			c.buffer.Next(1)

			continue
		}

		eject(pkt)
		c.buffer.Next(consumed)
	}

	if c.buffer.Len() > 0 {
		log.Warnf("Discarding %d stale serial bytes on idle timeout: 0x%x", c.buffer.Len(), c.buffer.Bytes())
		c.buffer.Reset()
	}
}

func NewChunker(deserializer packet.Deserializer, minLength int) Chunker {
	return &chunker{
		deserializer: deserializer,
		minLength:    minLength,
	}
}
