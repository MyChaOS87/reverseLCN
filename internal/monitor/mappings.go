//nolint:gochecknoglobals
package monitor

import "fmt"

var cmdMap = map[int]string{
	0x13: "relais",
	0x68: "statusReport",
	0x6E: "statusQuery",
}

var payloadParserByCommand = map[int]func(moduleOutputs map[int]map[int]string, src, dst int, payload []byte) string{
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

func mapOutputIfPossible(m map[int]map[int]string, module, output int) string {
	if outputs, ok := m[module]; ok {
		if s, ok := outputs[output]; ok {
			return s
		}
	}

	return fmt.Sprintf("%d-%d", module, output)
}
