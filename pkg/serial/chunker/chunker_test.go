package chunker_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

// test packet has a min length of 2 and max length of 3
// valid first byte is packet length, rest needs to be same as first byte.
type testPacket []byte

// Serialize implements packet.Packet.
func (t *testPacket) Serialize() ([]byte, error) {
	panic("unimplemented")
}

// ToNiceString implements packet.Packet.
func (*testPacket) ToNiceString() string {
	panic("unimplemented")
}

// ToString implements packet.Packet.
func (*testPacket) ToString() string {
	panic("unimplemented")
}

var _ packet.Packet = &testPacket{}

func testDeserialize(buf []byte) (packet.Packet, error) {
	if len(buf) < 2 {
		return nil, packet.ErrPacketInvalid
	}

	// first byte is the expected length of the packet, only 2 or 3 are valid
	expectedLenght := int(buf[0])
	if expectedLenght > 3 || expectedLenght < 2 {
		return nil, packet.ErrPacketInvalid
	}

	// if the buffer is not long enough, return incomplete
	if len(buf) < expectedLenght {
		return nil, packet.ErrPacketIncomplete
	}

	// check that all elements of buffer are the same
	for _, b := range buf[1:] {
		if b != buf[0] {
			return nil, packet.ErrPacketInvalid
		}
	}

	r := make(testPacket, 0, len(buf))
	r = append(r, buf...)

	return &r, nil
}

var _ packet.Deserializer = testDeserialize

func TestChunkerCollect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		buffers  [][]byte
		expected []packet.Packet
	}{
		{
			name:     "nothing",
			buffers:  [][]byte{},
			expected: nil,
		},
		{
			name:    "simple",
			buffers: [][]byte{{2, 2}},
			expected: []packet.Packet{
				&testPacket{2, 2},
			},
		},
		{
			name:    "simple * 2",
			buffers: [][]byte{{2, 2}, {2, 2}},
			expected: []packet.Packet{
				&testPacket{2, 2},
				&testPacket{2, 2},
			},
		},
		{
			name:    "simple partial",
			buffers: [][]byte{{2}, {2}},
			expected: []packet.Packet{
				&testPacket{2, 2},
			},
		},
		{
			name:    "simple partial + top level bs..",
			buffers: [][]byte{{2}, {2}, {4, 4}, {3, 3}, {3}},
			expected: []packet.Packet{
				&testPacket{2, 2},
				&testPacket{3, 3, 3},
			},
		},
		{
			name:    "wrong package search for valid one",
			buffers: [][]byte{{2}, {2}, {4, 4}, {3, 3}, {2}, {2}, {3, 3, 3}},
			expected: []packet.Packet{
				&testPacket{2, 2},
				&testPacket{2, 2},
				&testPacket{3, 3, 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			c := chunker.NewChunker(testDeserialize, 2)

			var got []packet.Packet

			for _, buf := range tt.buffers {
				c.Collect(buf, func(p packet.Packet) {
					got = append(got, p)
				})
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}
