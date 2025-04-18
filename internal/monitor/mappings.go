//nolint:gochecknoglobals
package monitor

import "fmt"

var idMap = map[int]string{
	4:  "Display",
	11: "LS Küche Eingang",
	12: "LS Küche Fenster",
	13: "LS Wohnzimmer",
	31: "R8H 31 - Jalousie A",
	32: "R8H 32 - Jalousie B",
	33: "R8H 33 - Licht A",
	34: "R8H 34 - Licht B",
	35: "SH 35 - Licht Dimmer",
}

var moduleOutputs = map[int]map[int]string{
	33: {
		0: "Strahler Wohnen/Essen",
		1: "Deckenlampe Wohnzimmertisch",
		2: "Wandlampe Wohnen Aussen", // unknown
		3: "Deckenlampe Wohnen West", // unknown
		5: "Aussenlicht Ost",
		6: "Aussenlicht Südwest",
		7: "Wandlampe Wohnzimmer Nord",
	},
	34: {
		0: "Strahler Küche x3",
		2: "Strahler Küche x5",
		4: "Aussenlicht Süd",
	},
	35: {
		0: "Deckenlampe Esszimmer",
		1: "Deckenlampe Küche",
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
