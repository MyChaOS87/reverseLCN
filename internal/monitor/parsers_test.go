package monitor_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MyChaOS87/reverseLCN/internal/monitor"
)

func syntheticIDMap() map[int]string {
	return map[int]string{
		4:  "Display",
		33: "R8H 33 - Relay A",
	}
}

func syntheticModuleOutputs() map[int]map[int]string {
	return map[int]map[int]string{
		33: {
			0: "Relay Living Room",
			1: "Light Living Room",
		},
		34: {
			0: "Light Kitchen 1",
			2: "Light Kitchen 2",
			4: "Light Corridor",
		},
		35: {
			0: "Dimmer Dining",
			1: "Dimmer Kitchen",
		},
	}
}

func TestParsePayloadIfPossible(t *testing.T) {
	t.Parallel()

	moduleOutputs := syntheticModuleOutputs()

	tests := []struct {
		name          string
		moduleOutputs map[int]map[int]string
		src           int
		dst           int
		cmd           int
		payload       []byte
		expected      string
	}{
		// 0x13: decodeRelais
		{
			name:          "relais mapped module force off",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{0b00000001, 0b00000001},
			expected:      "<Relay Living Room: FORCE OFF>",
		},
		{
			name:          "relais mapped module force on",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{0b00000010, 0b00000000},
			expected:      "<Light Living Room: FORCE ON>",
		},
		{
			name:          "relais mapped module toggle",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           34,
			cmd:           0x13,
			payload:       []byte{0b00000000, 0b00000001},
			expected:      "<Light Kitchen 1: TOGGLE>",
		},
		{
			name:          "relais multiple outputs mixed",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           35,
			cmd:           0x13,
			payload:       []byte{0b00000001, 0b00000010},
			expected:      "<Dimmer Dining: FORCE ON>,<Dimmer Kitchen: TOGGLE>",
		},
		{
			name:          "relais nil outputs map fallback",
			moduleOutputs: nil,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{0b00000001, 0b00000001},
			expected:      "<33-0: FORCE OFF>",
		},
		{
			name:          "relais unmapped module and unmapped output",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           99,
			cmd:           0x13,
			payload:       []byte{0b00000001, 0b00000001},
			expected:      "<99-0: FORCE OFF>",
		},
		{
			name:          "relais mapped module unmapped output bit 4",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{0b00010000, 0b00000000},
			expected:      "<33-4: FORCE ON>",
		},
		{
			name:          "relais nil payload fallback",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       nil,
			expected:      "",
		},
		{
			name:          "relais empty payload fallback",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{},
			expected:      "",
		},
		{
			name:          "relais short 1-byte payload fallback",
			moduleOutputs: moduleOutputs,
			src:           11,
			dst:           33,
			cmd:           0x13,
			payload:       []byte{0x01},
			expected:      "01",
		},

		// 0x68: decodeStatusReport
		{
			name:          "statusReport valid 0x30 dst 4 single output",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           4,
			cmd:           0x68,
			payload:       []byte{0x30, 0b00000011},
			expected:      "Relay Living Room,Light Living Room",
		},
		{
			name:          "statusReport valid 0x30 dst 4 zero active outputs",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           4,
			cmd:           0x68,
			payload:       []byte{0x30, 0x00},
			expected:      "",
		},
		{
			name:          "statusReport non-0x30 first byte fallback",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           4,
			cmd:           0x68,
			payload:       []byte{0x31, 0x01},
			expected:      "3101",
		},
		{
			name:          "statusReport dst not 4 fallback",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           11,
			cmd:           0x68,
			payload:       []byte{0x30, 0x01},
			expected:      "3001",
		},
		{
			name:          "statusReport nil payload fallback",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           4,
			cmd:           0x68,
			payload:       nil,
			expected:      "",
		},
		{
			name:          "statusReport short 1-byte payload fallback",
			moduleOutputs: moduleOutputs,
			src:           33,
			dst:           4,
			cmd:           0x68,
			payload:       []byte{0x30},
			expected:      "30",
		},

		// 0x6E: decodeStatusQuery
		{
			name:          "statusQuery 0xFB query uses dst module",
			moduleOutputs: moduleOutputs,
			src:           4,
			dst:           34,
			cmd:           0x6E,
			payload:       []byte{0xFB, 0b00010101},
			expected:      "QUERY: Light Kitchen 1,Light Kitchen 2,Light Corridor",
		},
		{
			name:          "statusQuery 0x7b report uses src module",
			moduleOutputs: moduleOutputs,
			src:           35,
			dst:           4,
			cmd:           0x6E,
			payload:       []byte{0x7b, 0b00000011},
			expected:      "REPORT: Dimmer Dining,Dimmer Kitchen",
		},
		{
			name:          "statusQuery unknown sub-command fallback",
			moduleOutputs: moduleOutputs,
			src:           35,
			dst:           4,
			cmd:           0x6E,
			payload:       []byte{0x00, 0x01},
			expected:      "0001",
		},
		{
			name:          "statusQuery nil payload fallback",
			moduleOutputs: moduleOutputs,
			src:           35,
			dst:           4,
			cmd:           0x6E,
			payload:       nil,
			expected:      "",
		},
		{
			name:          "statusQuery short 1-byte payload fallback",
			moduleOutputs: moduleOutputs,
			src:           35,
			dst:           4,
			cmd:           0x6E,
			payload:       []byte{0xFB},
			expected:      "fb",
		},

		// Unknown command fallback
		{
			name:          "unknown command default hex parser",
			moduleOutputs: moduleOutputs,
			src:           1,
			dst:           4,
			cmd:           0x99,
			payload:       []byte{0xde, 0xad, 0xbe, 0xef},
			expected:      "deadbeef",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			got := monitor.ParsePayloadIfPossible(tt.moduleOutputs, tt.src, tt.dst, tt.cmd, tt.payload)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDigitBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		b        byte
		out      int
		expected bool
	}{
		{
			name:     "negative index",
			b:        0xFF,
			out:      -1,
			expected: false,
		},
		{
			name:     "index 8 out of bounds",
			b:        0xFF,
			out:      8,
			expected: false,
		},
		{
			name:     "bit 0 set",
			b:        0b00000001,
			out:      0,
			expected: true,
		},
		{
			name:     "bit 0 unset",
			b:        0b11111110,
			out:      0,
			expected: false,
		},
		{
			name:     "bit 7 set",
			b:        0b10000000,
			out:      7,
			expected: true,
		},
		{
			name:     "bit 7 unset",
			b:        0b01111111,
			out:      7,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, monitor.CheckDigit(tt.b, tt.out))
		})
	}
}

func TestMappings(t *testing.T) {
	t.Parallel()

	idMap := syntheticIDMap()
	moduleOutputs := syntheticModuleOutputs()

	idTests := []struct {
		name     string
		idMap    map[int]string
		id       int
		expected string
	}{
		{name: "known Display", idMap: idMap, id: 4, expected: "Display"},
		{name: "known module 33", idMap: idMap, id: 33, expected: "R8H 33 - Relay A"},
		{name: "unknown module 99", idMap: idMap, id: 99, expected: "99"},
		{name: "nil map fallback", idMap: nil, id: 33, expected: "33"},
	}

	for _, tt := range idTests {
		t.Run(fmt.Sprintf("%s_id_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, monitor.MapIDIfPossible(tt.idMap, tt.id))
		})
	}

	cmdTests := []struct {
		name     string
		cmd      int
		expected string
	}{
		{name: "known relais", cmd: 0x13, expected: "relais"},
		{name: "known statusReport", cmd: 0x68, expected: "statusReport"},
		{name: "known statusQuery", cmd: 0x6E, expected: "statusQuery"},
		{name: "unknown cmd", cmd: 0x22, expected: fmt.Sprintf("%d", 0x22)},
	}

	for _, tt := range cmdTests {
		t.Run(fmt.Sprintf("%s_cmd_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, monitor.MapCmdIfPossible(tt.cmd))
		})
	}

	outTests := []struct {
		name          string
		moduleOutputs map[int]map[int]string
		module        int
		output        int
		expected      string
	}{
		{name: "mapped module and output", moduleOutputs: moduleOutputs, module: 33, output: 0, expected: "Relay Living Room"},
		{name: "mapped module unmapped output", moduleOutputs: moduleOutputs, module: 33, output: 4, expected: "33-4"},
		{name: "unmapped module", moduleOutputs: moduleOutputs, module: 99, output: 0, expected: "99-0"},
		{name: "nil map fallback", moduleOutputs: nil, module: 33, output: 0, expected: "33-0"},
	}

	for _, tt := range outTests {
		t.Run(fmt.Sprintf("%s_out_%s", t.Name(), tt.name), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, monitor.MapOutputIfPossible(tt.moduleOutputs, tt.module, tt.output))
		})
	}
}
