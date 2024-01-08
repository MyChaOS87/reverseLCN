package chunker_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
)

type ejectExpectation struct {
	add func(e *ejectMock) int
}

type ejectMock struct {
	mock.Mock
}

func (e *ejectMock) eject(p packet.Packet) {
	e.Called(p)
}

func onEject(p packet.Packet, times int) ejectExpectation {
	return ejectExpectation{
		add: func(e *ejectMock) int {
			e.On("eject", p).Times(times)

			return times
		},
	}
}

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

//nolint:funlen
func TestChunkerCollect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		buffers           [][]byte
		ejectExpectations []ejectExpectation
	}{
		{
			name:              "nothing",
			buffers:           [][]byte{},
			ejectExpectations: []ejectExpectation{},
		},
		{
			name:    "simple",
			buffers: [][]byte{{2, 2}},
			ejectExpectations: []ejectExpectation{
				onEject(&testPacket{2, 2}, 1),
			},
		},
		{
			name:    "simple * 2",
			buffers: [][]byte{{2, 2}, {2, 2}},
			ejectExpectations: []ejectExpectation{
				onEject(&testPacket{2, 2}, 2),
			},
		},
		{
			name:    "simple partial",
			buffers: [][]byte{{2}, {2}},
			ejectExpectations: []ejectExpectation{
				onEject(&testPacket{2, 2}, 1),
			},
		},
		{
			name:    "simple partial + top level bs..",
			buffers: [][]byte{{2}, {2}, {4, 4}, {3, 3}, {3}},
			ejectExpectations: []ejectExpectation{
				onEject(&testPacket{2, 2}, 1),
				onEject(&testPacket{3, 3, 3}, 1),
			},
		},
		{
			name:    "wrong package search for valid one",
			buffers: [][]byte{{2}, {2}, {4, 4}, {3, 3}, {2}, {2}, {3, 3, 3}},
			ejectExpectations: []ejectExpectation{
				onEject(&testPacket{2, 2}, 2),
				onEject(&testPacket{3, 3, 3}, 1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			c := chunker.NewChunker(testDeserialize, 2)

			e := new(ejectMock)

			expectedEjectCalls := 0
			for _, ejectExpectation := range tt.ejectExpectations {
				expectedEjectCalls += ejectExpectation.add(e)
			}

			for _, buf := range tt.buffers {
				c.Collect(buf, e.eject)
			}

			e.AssertExpectations(t)
			e.AssertNumberOfCalls(t, "eject", expectedEjectCalls)
		})
	}
}
