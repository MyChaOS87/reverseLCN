package plain

import (
	"encoding/hex"

	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

type Plain []byte

func Deserialize(buf []byte) (packet.Packet, error) {
	p := make(Plain, len(buf))
	copy(p, buf)

	return &p, nil
}

func (p *Plain) Serialize() ([]byte, error) {
	return []byte(p.ToString()), nil
}

func (p *Plain) ToString() string {
	return hex.EncodeToString(*p)
}

func (p *Plain) ToNiceString() string {
	return p.ToString()
}
