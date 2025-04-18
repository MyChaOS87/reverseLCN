//nolint:gochecknoglobals
package monitor

import "fmt"

var idMap = map[int]string{
	4:  "Display",
	11: "Switch Kitchen",
	12: "Switch Kitchen 2",
	13: "Switch Living Room",
	31: "R8H 31 - Shutter A",
	32: "R8H 32 - Shutter B",
	33: "R8H 33 - Relay A",
	34: "R8H 34 - Relay B",
	35: "SH 35 - Dimmer",
}

var moduleOutputs = map[int]map[int]string{
	33: {
		0: "Relay Living Room",
		1: "Light Living Room 1",
		2: "Light Living Room 2", // unknown
		3: "Light Living Room 3", // unknown
		5: "Light Auxiliary 1",
		6: "Light Auxiliary 2",
		7: "Light Living Room 4",
	},
	34: {
		0: "Light Kitchen 1",
		2: "Light Kitchen 2",
		4: "Light Auxiliary 3",
	},
	35: {
		0: "Dimmer Dining",
		1: "Dimmer Kitchen",
	},
}

var cmdMap = map[int]string{
	0x13: "relais",
	0x68: "statusReport",
	0x6E: "statusQuery",
}

var payloadParserByCommand = map[int]func(src, dst int, payload []byte) string{
	0x13: decodeRelais,
	0x68: decodeStatusReport,
	0x6E: decodeStatusQuery,
}

func mapIfPossible(m map[int]string, value int) string {
	if s, ok := m[value]; ok {
		return s
	}

	return fmt.Sprintf("%d", value)
}

func mapOutputIfPossible(module int, output int) string {
	if m, ok := moduleOutputs[module]; ok {
		if s, ok := m[output]; ok {
			return s
		}
	}

	return fmt.Sprintf("%d-%d", module, output)
}
