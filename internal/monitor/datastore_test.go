package monitor_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/reverseLCN/internal/monitor"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
)

func TestDataStoreAddAndQuery(t *testing.T) {
	t.Parallel()

	pktRelais := lcn.Packet{
		Src:     11,
		Seg:     0,
		Dst:     33,
		Cmd:     0x13,
		Payload: []byte{0x01, 0x01},
	}
	pktStatus := lcn.Packet{
		Src:     33,
		Seg:     0,
		Dst:     4,
		Cmd:     0x68,
		Payload: []byte{0x30, 0x01},
	}

	tests := []struct {
		name           string
		packets        []lcn.Packet
		queryLimit     int
		expectedCount  int
		expectedTimes  []int
		expectedSuffix string
	}{
		{
			name:          "empty datastore",
			packets:       nil,
			queryLimit:    5,
			expectedCount: 0,
			expectedTimes: nil,
		},
		{
			name:           "single insert",
			packets:        []lcn.Packet{pktRelais},
			queryLimit:     5,
			expectedCount:  1,
			expectedTimes:  []int{1},
			expectedSuffix: "1\tLS Küche Eingang\t0\tR8H 33 - Licht A\trelais\t<Strahler Wohnen/Essen: FORCE OFF>",
		},
		{
			name:           "duplicate insert increments times",
			packets:        []lcn.Packet{pktRelais, pktRelais, pktRelais},
			queryLimit:     5,
			expectedCount:  1,
			expectedTimes:  []int{3},
			expectedSuffix: "3\tLS Küche Eingang\t0\tR8H 33 - Licht A\trelais\t<Strahler Wohnen/Essen: FORCE OFF>",
		},
		{
			name:          "multiple distinct packets with limit",
			packets:       []lcn.Packet{pktRelais, pktStatus, pktRelais},
			queryLimit:    1,
			expectedCount: 1,
			expectedTimes: []int{2},
		},
		{
			name:          "multiple distinct packets all returned",
			packets:       []lcn.Packet{pktRelais, pktStatus},
			queryLimit:    10,
			expectedCount: 2,
			expectedTimes: []int{1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			ds := monitor.NewDataStore()
			require.NotNil(t, ds)

			for _, p := range tt.packets {
				ds.Add(p)
			}

			got := ds.GetLast(tt.queryLimit)
			require.Len(t, got, tt.expectedCount)

			for i, expTimes := range tt.expectedTimes {
				assert.Equal(t, expTimes, got[i].Times)
			}

			if tt.expectedSuffix != "" {
				assert.True(t, strings.HasSuffix(got[0].Formatted, tt.expectedSuffix),
					"expected suffix %q in %q", tt.expectedSuffix, got[0].Formatted)
			}
		})
	}
}

func TestDataStoreRenderData(t *testing.T) {
	t.Parallel()

	pkt1 := lcn.Packet{
		Src:     11,
		Seg:     0,
		Dst:     33,
		Cmd:     0x13,
		Payload: []byte{0x01, 0x00},
	}
	pkt2 := lcn.Packet{
		Src:     35,
		Seg:     0,
		Dst:     4,
		Cmd:     0x6E,
		Payload: []byte{0x7b, 0x01},
	}

	ds := monitor.NewDataStore()
	ds.Add(pkt1)
	ds.Add(pkt2)
	ds.Add(pkt1)

	table := ds.RenderData()
	require.Len(t, table, 3)
	assert.Equal(t, []string{"last seen", "#", "Src", "Seg", "Dst", "Command", "Payload"}, table[0])

	// Verify both rendered rows contain 7 columns
	for i := 1; i < len(table); i++ {
		assert.Len(t, table[i], 7)
	}
}

func TestDataStoreShortPayloadsNoPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		packet lcn.Packet
	}{
		{
			name:   "relais empty payload",
			packet: lcn.Packet{Src: 11, Dst: 33, Cmd: 0x13, Payload: nil},
		},
		{
			name:   "relais 1-byte payload",
			packet: lcn.Packet{Src: 11, Dst: 33, Cmd: 0x13, Payload: []byte{0x01}},
		},
		{
			name:   "statusReport empty payload",
			packet: lcn.Packet{Src: 33, Dst: 4, Cmd: 0x68, Payload: []byte{}},
		},
		{
			name:   "statusReport 1-byte payload",
			packet: lcn.Packet{Src: 33, Dst: 4, Cmd: 0x68, Payload: []byte{0x30}},
		},
		{
			name:   "statusQuery empty payload",
			packet: lcn.Packet{Src: 4, Dst: 34, Cmd: 0x6E, Payload: nil},
		},
		{
			name:   "statusQuery 1-byte payload",
			packet: lcn.Packet{Src: 4, Dst: 34, Cmd: 0x6E, Payload: []byte{0xFB}},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			ds := monitor.NewDataStore()

			require.NotPanics(t, func() {
				ds.Add(tt.packet)
			})

			got := ds.GetLast(1)
			require.Len(t, got, 1)
			assert.Equal(t, 1, got[0].Times)
		})
	}
}
