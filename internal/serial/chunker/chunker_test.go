package chunker_test

import (
	"fmt"
	"testing"

	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/packet"
	"github.com/stretchr/testify/mock"
)

type ejectExpectation struct {
	add func(e *ejectMock) int
}

type ejectMock struct {
	mock.Mock
}

func (e *ejectMock) eject(p packet.Packet) {
	e.Called(p.Serialize())
}

func onEject(b []byte, times int) ejectExpectation {
	return ejectExpectation{
		add: func(e *ejectMock) int {
			e.On("eject", b).Times(times)
			return times
		},
	}
}

// test packet has a min length of 2 and max length of 3
// valid first byte is packet length, rest is arbitrary payload
type testPacket []byte

// Serialize implements packet.Packet.
func (t *testPacket) Serialize() []byte {
	return *t
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
	expectedLenght := int(buf[0])
	if expectedLenght > 3 || expectedLenght < 2 {
		return nil, packet.ErrPacketInvalid
	}
	if len(buf) < expectedLenght {
		return nil, packet.ErrPacketIncomplete
	}

	if expectedLenght != len(buf) {
		return nil, packet.ErrPacketInvalid
	}

	r := make(testPacket, 0, len(buf))
	r = append(r, buf...)
	return &r, nil
}

var _ packet.Deserializer = testDeserialize

func TestChunkerCollect(t *testing.T) {
	tests := []struct {
		name              string
		buffers           [][]byte
		ejectExpectations []ejectExpectation
	}{
		{name: "nothing", buffers: [][]byte{}, ejectExpectations: []ejectExpectation{}},
		{
			name: "simple", buffers: [][]byte{{2, 2}},
			ejectExpectations: []ejectExpectation{
				onEject([]byte{2, 2}, 1),
			},
		},
		{
			name: "simple * 2", buffers: [][]byte{{2, 2}, {2, 2}},
			ejectExpectations: []ejectExpectation{
				onEject([]byte{2, 2}, 2),
			},
		},
		{
			name: "simple partial", buffers: [][]byte{{2}, {2}},
			ejectExpectations: []ejectExpectation{
				onEject([]byte{2, 2}, 1),
			},
		},
		{
			name: "simple partial + top level bs..", buffers: [][]byte{{2}, {3}, {4, 5}, {3, 4}, {5}},
			ejectExpectations: []ejectExpectation{
				onEject([]byte{2, 3}, 1),
				onEject([]byte{3, 4, 5}, 1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			c := chunker.NewChunker(testDeserialize, 2)

			e := new(ejectMock)

			expectedEjectCalls := 0
			for _, ejectExpectation := range tt.ejectExpectations {
				expectedEjectCalls += ejectExpectation.add(e)
			}

			for _, buf := range tt.buffers {
				c.Collect([]byte(buf), e.eject)
			}

			e.AssertExpectations(t)
			e.AssertNumberOfCalls(t, "eject", expectedEjectCalls)
		})
	}
}
