package lcn_test

import (
	"fmt"
	"testing"

	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/packet"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	assert.ErrorIs(t, lcn.ErrLcnPacketIncomplete, packet.ErrPacketIncomplete)
	assert.ErrorIs(t, lcn.ErrLcnPacketInvalid, packet.ErrPacketInvalid)
	assert.ErrorIs(t, lcn.ErrLcnPacketInvalidChecksum, packet.ErrPacketInvalid)
}

func TestDeserialize(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		error  error
		packet packet.Packet
	}{
		{
			name:  "empty",
			input: []byte{},
			error: lcn.ErrLcnPacketIncomplete,
		},
		// There is no check yet if cmd 5 is actually valid for length 6 through 20 as we have too little information atm
		{
			name:   "synthetic length 6",
			input:  []byte{0x80, 0x00, 0xd5, 0x2, 0x4, 0x5},
			packet: &lcn.LcnPacket{Src: 1, Info: 0, Checksum: 213, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
		},
		{
			name:   "synthetic length 8",
			input:  []byte{0x80, 0b01 << 2, 0x15, 0x2, 0x4, 0x5, 0x06, 0x07},
			packet: &lcn.LcnPacket{Src: 1, Info: 0b01 << 2, Checksum: 21, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7}},
		},
		{
			name:   "synthetic length 12",
			input:  []byte{0x80, 0b10 << 2, 0x26, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B},
			packet: &lcn.LcnPacket{Src: 1, Info: 0b10 << 2, Checksum: 38, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7, 8, 9, 10, 11}},
		},
		{
			name:   "synthetic length 20",
			input:  []byte{0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13},
			packet: &lcn.LcnPacket{Src: 1, Info: 0b11 << 2, Checksum: 65, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}},
		},
		{
			name:  "synthetic length 8 too short",
			input: []byte{0x80, 0b01 << 2, 0x15, 0x2, 0x4, 0x5, 0x06},
			error: lcn.ErrLcnPacketIncomplete,
		},
		{
			name:  "synthetic length 12 too short",
			input: []byte{0x80, 0b10 << 2, 0x26, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A},
			error: lcn.ErrLcnPacketIncomplete,
		},
		{
			name:  "synthetic length 20 too short",
			input: []byte{0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12},
			error: lcn.ErrLcnPacketIncomplete,
		},
		{
			name:   "real life 8",
			input:  []byte{0xa8, 0x06, 0x75, 0x00, 0x04, 0x68, 0x30, 0x00},
			packet: &lcn.LcnPacket{Src: 0x15, Info: 0x6, Checksum: 0x75, Seg: 0x0, Dst: 0x4, Cmd: 0x68, Payload: []uint8{0x30, 0x0}},
		},
		{
			name:   "real life 20",
			input:  []byte{0xf8, 0x4e, 0x66, 0x04, 0x04, 0x22, 0x01, 0x00, 0x05, 0x38, 0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41},
			packet: &lcn.LcnPacket{Src: 0x1f, Info: 0x4e, Checksum: 0x66, Seg: 0x4, Dst: 0x4, Cmd: 0x22, Payload: []uint8{0x1, 0x0, 0x5, 0x38, 0x13, 0x3, 0xb, 0x17, 0x5, 0x3c, 0x0, 0x0, 0x1, 0x41}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			pkt, err := lcn.Deserialize(tt.input)
			if tt.error == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.packet, pkt)
			} else {
				assert.ErrorIs(t, tt.error, err)
				assert.Nil(t, pkt)
			}
		})
	}
}
