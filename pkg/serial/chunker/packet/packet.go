package packet

import "errors"

var (
	ErrPacketIncomplete = errors.New("Packet Incomplete")
	ErrPacketInvalid    = errors.New("Packet Invalid")
)

type Packet interface {
	Serialize() []byte
	ToString() string     // should return all embedded information
	ToNiceString() string // can leave out less relevant information and should highlight most important information
}

type Deserializer func(buf []byte) (Packet, error)
