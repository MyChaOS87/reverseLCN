package lcn_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, lcn.ErrLcnPacketIncomplete, packet.ErrPacketIncomplete)
	require.ErrorIs(t, lcn.ErrLcnPacketInvalid, packet.ErrPacketInvalid)
	require.ErrorIs(t, lcn.ErrLcnPacketInvalidChecksum, packet.ErrPacketInvalid)
}

func TestDeserialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		error    error
		consumed int
		packet   packet.Packet
	}{
		{
			name:  "empty",
			input: []byte{},
			error: lcn.ErrLcnPacketIncomplete,
		},
		// There is no check yet if cmd 5 is actually valid for length 6 through 20 as we have too little information atm
		{
			name:     "synthetic length 6",
			input:    []byte{0x80, 0x00, 0xd5, 0x2, 0x4, 0x5},
			consumed: 6,
			packet:   &lcn.Packet{Src: 1, Info: 0, Checksum: 213, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
		},
		{
			name:     "synthetic length 8",
			input:    []byte{0x80, 0b01 << 2, 0x15, 0x2, 0x4, 0x5, 0x06, 0x07},
			consumed: 8,
			packet:   &lcn.Packet{Src: 1, Info: 0b01 << 2, Checksum: 21, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7}},
		},
		{
			name:     "synthetic length 12",
			input:    []byte{0x80, 0b10 << 2, 0x26, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B},
			consumed: 12,
			packet: &lcn.Packet{
				Src: 1, Info: 0b10 << 2, Checksum: 38, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11},
			},
		},
		{
			name: "synthetic length 20",
			input: []byte{
				0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09,
				0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13,
			},
			consumed: 20,
			packet: &lcn.Packet{
				Src: 1, Info: 0b11 << 2, Checksum: 65, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			},
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
			name: "synthetic length 20 too short",
			input: []byte{
				0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09,
				0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12,
			},
			error: lcn.ErrLcnPacketIncomplete,
		},
		{
			name:     "real life 8",
			input:    []byte{0xa8, 0x06, 0x75, 0x00, 0x04, 0x68, 0x30, 0x00},
			consumed: 8,
			packet:   &lcn.Packet{Src: 0x15, Info: 0x6, Checksum: 0x75, Seg: 0x0, Dst: 0x4, Cmd: 0x68, Payload: []uint8{0x30, 0x0}},
		},
		{
			name: "real life 20",
			input: []byte{
				0xf8, 0x4e, 0x66, 0x04, 0x04, 0x22, 0x01, 0x00, 0x05, 0x38,
				0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41,
			},
			consumed: 20,
			packet: &lcn.Packet{
				Src: 0x1f, Info: 0x4e, Checksum: 0x66, Seg: 0x4, Dst: 0x4, Cmd: 0x22,
				Payload: []uint8{0x1, 0x0, 0x5, 0x38, 0x13, 0x3, 0xb, 0x17, 0x5, 0x3c, 0x0, 0x0, 0x1, 0x41},
			},
		},
		{
			name:  "corrupted checksum length 6",
			input: []byte{0x80, 0x00, 0xd4, 0x2, 0x4, 0x5},
			error: lcn.ErrLcnPacketInvalidChecksum,
		},
		{
			name:  "corrupted checksum length 8",
			input: []byte{0xa8, 0x06, 0x00, 0x00, 0x04, 0x68, 0x30, 0x00},
			error: lcn.ErrLcnPacketInvalidChecksum,
		},
		{
			name:     "buffer with trailing bytes length 6",
			input:    []byte{0x80, 0x00, 0xd5, 0x2, 0x4, 0x5, 0x99},
			consumed: 6,
			packet:   &lcn.Packet{Src: 1, Info: 0, Checksum: 213, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
		},
		{
			name:     "buffer with trailing bytes length 8",
			input:    []byte{0x80, 0b01 << 2, 0x15, 0x2, 0x4, 0x5, 0x06, 0x07, 0x99},
			consumed: 8,
			packet:   &lcn.Packet{Src: 1, Info: 0b01 << 2, Checksum: 21, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7}},
		},
		{
			name:     "buffer with trailing bytes length 12",
			input:    []byte{0x80, 0b10 << 2, 0x26, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x99},
			consumed: 12,
			packet: &lcn.Packet{
				Src: 1, Info: 0b10 << 2, Checksum: 38, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11},
			},
		},
		{
			name: "buffer with trailing bytes length 20",
			input: []byte{
				0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09,
				0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x99,
			},
			consumed: 20,
			packet: &lcn.Packet{
				Src: 1, Info: 0b11 << 2, Checksum: 65, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			pkt, consumed, err := lcn.Deserialize(tt.input)
			assert.Equal(t, tt.consumed, consumed)

			if tt.error == nil {
				require.NoError(t, err)
				assert.Equal(t, tt.packet, pkt)
			} else {
				require.ErrorIs(t, err, tt.error)
				assert.Nil(t, pkt)
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		packet packet.Packet
		error  error
		output []byte
	}{
		// There is no check yet if cmd 5 is actually valid for length 6 through 20 as we have too little information atm
		{
			name:   "synthetic length 6",
			packet: &lcn.Packet{Src: 1, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
			output: []byte{0x80, 0x00, 0xd5, 0x2, 0x4, 0x5},
		},
		{
			name:   "synthetic length 8",
			packet: &lcn.Packet{Src: 1, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7}},
			output: []byte{0x80, 0b01 << 2, 0x15, 0x2, 0x4, 0x5, 0x06, 0x07},
		},
		{
			name:   "synthetic length 12",
			packet: &lcn.Packet{Src: 1, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6, 7, 8, 9, 10, 11}},
			output: []byte{0x80, 0b10 << 2, 0x26, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B},
		},
		{
			name: "synthetic length 20",
			packet: &lcn.Packet{
				Src: 1, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			},
			output: []byte{
				0x80, 0b11 << 2, 0x41, 0x2, 0x4, 0x5, 0x06, 0x07, 0x08, 0x09,
				0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13,
			},
		},

		{
			name:   "real life 8",
			packet: &lcn.Packet{Src: 0x15, Info: 0x2, Seg: 0x0, Dst: 0x4, Cmd: 0x68, Payload: []uint8{0x30, 0x0}},
			output: []byte{0xa8, 0x06, 0x75, 0x00, 0x04, 0x68, 0x30, 0x00},
		},
		{
			name: "real life 20",
			packet: &lcn.Packet{
				Src: 0x1f, Info: 0x42, Seg: 0x4, Dst: 0x4, Cmd: 0x22,
				Payload: []uint8{0x1, 0x0, 0x5, 0x38, 0x13, 0x3, 0xb, 0x17, 0x5, 0x3c, 0x0, 0x0, 0x1, 0x41},
			},
			output: []byte{
				0xf8, 0x4e, 0x66, 0x04, 0x04, 0x22, 0x01, 0x00, 0x05, 0x38,
				0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41,
			},
		},
		{
			name:   "strange length",
			packet: &lcn.Packet{Src: 1, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{6}},
			error:  lcn.ErrLcnPacketInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			output, err := tt.packet.Serialize()
			if tt.error == nil {
				require.NoError(t, err)
				assert.Equal(t, tt.output, output)
			} else {
				require.ErrorIs(t, err, tt.error)
				assert.Nil(t, output)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *lcn.Packet
		expected *lcn.Packet
	}{
		{
			name:     "frame length 6",
			input:    &lcn.Packet{Src: 1, Info: 0x00, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
			expected: &lcn.Packet{Src: 1, Info: 0x00, Checksum: 0xd5, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}},
		},
		{
			name: "frame length 8",
			input: &lcn.Packet{
				Src: 0x15, Info: 0x02, Seg: 0x00, Dst: 0x04, Cmd: 0x68,
				Payload: []byte{0x30, 0x00},
			},
			expected: &lcn.Packet{
				Src: 0x15, Info: 0x06, Checksum: 0x75, Seg: 0x00, Dst: 0x04, Cmd: 0x68,
				Payload: []byte{0x30, 0x00},
			},
		},
		{
			name: "frame length 12",
			input: &lcn.Packet{
				Src: 1, Info: 0x00, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11},
			},
			expected: &lcn.Packet{
				Src: 1, Info: 0b10 << 2, Checksum: 0x26, Seg: 2, Dst: 4, Cmd: 5,
				Payload: []byte{6, 7, 8, 9, 10, 11},
			},
		},
		{
			name: "frame length 20",
			input: &lcn.Packet{
				Src: 0x1f, Info: 0x42, Seg: 0x04, Dst: 0x04, Cmd: 0x22,
				Payload: []byte{0x01, 0x00, 0x05, 0x38, 0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41},
			},
			expected: &lcn.Packet{
				Src: 0x1f, Info: 0x4e, Checksum: 0x66, Seg: 0x04, Dst: 0x04, Cmd: 0x22,
				Payload: []byte{0x01, 0x00, 0x05, 0x38, 0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			raw, err := tt.input.Serialize()
			require.NoError(t, err)

			got, consumed, err := lcn.Deserialize(raw)
			require.NoError(t, err)
			assert.Equal(t, len(raw), consumed)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPacketFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		packet       *lcn.Packet
		expectedStr  string
		expectedNice string
	}{
		{
			name: "empty payload length 6",
			packet: &lcn.Packet{
				Src: 0x01, Info: 0x00, Checksum: 0xd5, Seg: 0x02, Dst: 0x04, Cmd: 0x05,
				Payload: []byte{},
			},
			expectedStr:  "src: 1, info: 0, crc: d5, seg: 2, dst: 4, cmd: 5, payload: ",
			expectedNice: " 1-> 2: 4 cmd:  5, payload: ",
		},
		{
			name: "status report length 8",
			packet: &lcn.Packet{
				Src: 0x15, Info: 0x06, Checksum: 0x75, Seg: 0x00, Dst: 0x04, Cmd: 0x68,
				Payload: []byte{0x30, 0x00},
			},
			expectedStr:  "src: 15, info: 6, crc: 75, seg: 0, dst: 4, cmd: 68, payload: 3000",
			expectedNice: "15-> 0: 4 cmd: 68, payload: 3000",
		},
		{
			name: "multi-byte payload length 20",
			packet: &lcn.Packet{
				Src: 0x1f, Info: 0x4e, Checksum: 0x66, Seg: 0x04, Dst: 0x04, Cmd: 0x22,
				Payload: []byte{0x01, 0x00, 0x05, 0x38, 0x13, 0x03, 0x0b, 0x17, 0x05, 0x3c, 0x00, 0x00, 0x01, 0x41},
			},
			expectedStr:  "src: 1f, info: 4e, crc: 66, seg: 4, dst: 4, cmd: 22, payload: 0100053813030b17053c00000141",
			expectedNice: "1f-> 4: 4 cmd: 22, payload: 0100053813030b17053c00000141",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedStr, tt.packet.ToString())
			assert.Equal(t, tt.expectedNice, tt.packet.ToNiceString())
		})
	}
}

func TestChunkerLCNNoiseRecovery(t *testing.T) {
	t.Parallel()

	// Src 48 (0x30) mirrors to wire byte 0x0C, which sets both length bits (0x0C) so
	// reading it at offset 1 masquerades as a 20-byte frame when preceded by 1 noise byte.
	pktA := &lcn.Packet{Src: 48, Info: 0x00, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}}
	wireA, err := pktA.Serialize()
	require.NoError(t, err)
	require.Equal(t, byte(0x0C), wireA[0])

	pktB := &lcn.Packet{Src: 1, Info: 0x00, Seg: 2, Dst: 4, Cmd: 5, Payload: []byte{}}
	wireB, err := pktB.Serialize()
	require.NoError(t, err)

	pktC := &lcn.Packet{Src: 0x15, Info: 0x02, Seg: 0x00, Dst: 0x04, Cmd: 0x68, Payload: []byte{0x30, 0x00}}
	wireC, err := pktC.Serialize()
	require.NoError(t, err)

	wantA, _, err := lcn.Deserialize(wireA)
	require.NoError(t, err)
	wantB, _, err := lcn.Deserialize(wireB)
	require.NoError(t, err)
	wantC, _, err := lcn.Deserialize(wireC)
	require.NoError(t, err)

	streamWithSubsequent := make([]byte, 0, 1+len(wireA)+len(wireB)+len(wireC))
	streamWithSubsequent = append(streamWithSubsequent, 0xFF)
	streamWithSubsequent = append(streamWithSubsequent, wireA...)
	streamWithSubsequent = append(streamWithSubsequent, wireB...)
	streamWithSubsequent = append(streamWithSubsequent, wireC...)

	quietBusStream := make([]byte, 0, 1+len(wireA))
	quietBusStream = append(quietBusStream, 0xFF)
	quietBusStream = append(quietBusStream, wireA...)

	tests := []struct {
		name     string
		chunks   [][]byte
		flush    bool
		trailing []byte
		expected []packet.Packet
	}{
		{
			name:     "1 noise byte + 6-byte frame with 0x0C wire Src + subsequent frames recovers all packets",
			chunks:   [][]byte{streamWithSubsequent},
			expected: []packet.Packet{wantA, wantB, wantC},
		},
		{
			name:     "1 noise byte + 6-byte frame with 0x0C wire Src stalled on quiet bus recovers on Flush",
			chunks:   [][]byte{quietBusStream},
			flush:    true,
			trailing: wireB,
			expected: []packet.Packet{wantA, wantB},
		},
		{
			name:     "sub-minLength noise cleared by Flush allows clean next frame",
			chunks:   [][]byte{{0xFF, 0xAA, 0x55}},
			flush:    true,
			trailing: wireC,
			expected: []packet.Packet{wantC},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := chunker.NewChunker(lcn.Deserialize, lcn.MinPacketLength)

			var got []packet.Packet

			eject := func(p packet.Packet) {
				got = append(got, p)
			}

			for _, chunk := range tt.chunks {
				c.Collect(chunk, eject)
			}

			if tt.flush {
				c.Flush(eject)
			}

			if len(tt.trailing) > 0 {
				c.Collect(tt.trailing, eject)
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}
