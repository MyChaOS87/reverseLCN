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

func testDeserialize(buf []byte) (packet.Packet, int, error) {
	if len(buf) < 2 {
		return nil, 0, packet.ErrPacketInvalid
	}

	// first byte is the expected length of the packet, only 2 or 3 are valid
	expectedLength := int(buf[0])
	if expectedLength > 3 || expectedLength < 2 {
		return nil, 0, packet.ErrPacketInvalid
	}

	// if the buffer is not long enough, return incomplete
	if len(buf) < expectedLength {
		return nil, 0, packet.ErrPacketIncomplete
	}

	frame := buf[:expectedLength]

	// check that all elements of the frame are the same
	for _, b := range frame[1:] {
		if b != frame[0] {
			return nil, 0, packet.ErrPacketInvalid
		}
	}

	r := make(testPacket, len(frame))
	copy(r, frame)

	return &r, expectedLength, nil
}

// headerLenDeserialize models protocols whose length field sits at buf[1] (like LCN's Info byte):
// frame = [tag, expectedLen, payload...], valid when expectedLen is 2 or 6 and all trailing bytes equal tag^expectedLen.
func headerLenDeserialize(buf []byte) (packet.Packet, int, error) {
	const (
		minLen  = 2
		longLen = 6
	)

	if len(buf) < minLen {
		return nil, 0, packet.ErrPacketIncomplete
	}

	expectedLen := int(buf[1])
	if expectedLen != minLen && expectedLen != longLen {
		return nil, 0, packet.ErrPacketInvalid
	}

	if len(buf) < expectedLen {
		return nil, 0, packet.ErrPacketIncomplete
	}

	frame := buf[:expectedLen]
	check := frame[0] ^ frame[1]

	for _, b := range frame[minLen:] {
		if b != check {
			return nil, 0, packet.ErrPacketInvalid
		}
	}

	r := make(testPacket, len(frame))
	copy(r, frame)

	return &r, expectedLen, nil
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

func TestChunkerResyncAndFlush(t *testing.T) {
	t.Parallel()

	type step struct {
		collect []byte
		flush   bool
	}

	tests := []struct {
		name     string
		steps    []step
		expected []packet.Packet
	}{
		{
			name: "1 noise byte before 2-byte frame whose first byte looks like 6-byte length recovers both frames",
			steps: []step{
				// 0xFF (noise) + V=[0x06, 0x02] + W1=[0x02, 0x02] + W2=[0x03, 0x02]
				// When 0xFF is at index 0, buf[1]==0x06 requests 6 bytes, accumulating 7 bytes across V+W1+W2.
				// Deserializer must consume only 2 bytes per valid frame via Next(consumed) instead of Reset().
				{collect: []byte{0xFF, 0x06, 0x02, 0x02, 0x02, 0x03, 0x02}},
			},
			expected: []packet.Packet{
				&testPacket{0x06, 0x02},
				&testPacket{0x02, 0x02},
				&testPacket{0x03, 0x02},
			},
		},
		{
			name: "idle flush recovers stalled shorter frame behind noise byte",
			steps: []step{
				{collect: []byte{0xFF, 0x06, 0x02}},
				{flush: true},
				{collect: []byte{0x05, 0x02}},
			},
			expected: []packet.Packet{
				&testPacket{0x06, 0x02},
				&testPacket{0x05, 0x02},
			},
		},
		{
			name: "idle flush discards sub-minLength noise fragment before next frame",
			steps: []step{
				{collect: []byte{0xAA}},
				{flush: true},
				{collect: []byte{0x04, 0x02}},
			},
			expected: []packet.Packet{
				&testPacket{0x04, 0x02},
			},
		},
		{
			name: "idle flush on empty buffer is a no-op",
			steps: []step{
				{flush: true},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := chunker.NewChunker(headerLenDeserialize, 2)

			var got []packet.Packet

			eject := func(p packet.Packet) {
				got = append(got, p)
			}

			for _, s := range tt.steps {
				if len(s.collect) > 0 {
					c.Collect(s.collect, eject)
				}

				if s.flush {
					c.Flush(eject)
				}
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}
