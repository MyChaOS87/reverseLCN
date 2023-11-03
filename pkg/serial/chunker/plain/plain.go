package plain

import (
	"encoding/hex"

	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
)

type PlainPacket []byte

func Deserialize(buf []byte) (packet.Packet, error) {
	p := make(PlainPacket, len(buf))
	copy(p, buf)
	return &p, nil
}

func (p *PlainPacket) Serialize() ([]byte, error) {
	return []byte(p.ToString()), nil
}

func (p *PlainPacket) ToString() string {
	return hex.EncodeToString(*p)
}

func (p *PlainPacket) ToNiceString() string {
	return p.ToString()
}
