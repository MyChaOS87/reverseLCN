package plain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/plain"
)

func TestPlainPacket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       []byte
		expectedHex string
	}{
		{
			name:        "binary frame round-trips raw bytes and formats hex",
			input:       []byte{0xde, 0xad, 0xbe, 0xef},
			expectedHex: "deadbeef",
		},
		{
			name:        "empty byte slice round-trips cleanly",
			input:       []byte{},
			expectedHex: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkt, err := plain.Deserialize(tt.input)
			require.NoError(t, err)
			require.NotNil(t, pkt)

			serialized, err := pkt.Serialize()
			require.NoError(t, err)
			assert.Equal(t, tt.input, serialized)
			assert.Equal(t, tt.expectedHex, pkt.ToString())
			assert.Equal(t, tt.expectedHex, pkt.ToNiceString())
		})
	}
}
