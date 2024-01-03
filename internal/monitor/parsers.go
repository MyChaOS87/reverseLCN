package monitor

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func defaultPayloadParser(_, _ int, payload []byte) string {
	return hex.EncodeToString(payload)
}

func parsePayloadIfPossible(src, dst, cmd int, payload []byte) string {
	if f, ok := payloadParserByCommand[cmd]; ok {
		return f(src, dst, payload)
	}

	return defaultPayloadParser(src, dst, payload)
}

func testDigit(b byte, out int) bool {
	return b&(1<<uint(out)) != 0
}

func decodeSwitch(src, dst int, payload []byte) string {
	outputs := make([]string, 0)

	for i := 0; i < 8; i++ {
		bitPositions := payload[0] | payload[1]

		if testDigit(bitPositions, i) {
			outputName := mapOutputIfPossible(dst, i)
			force := testDigit(payload[0], i)
			toggle := testDigit(payload[1], i)

			switch {
			case force && toggle:
				outputs = append(outputs, fmt.Sprintf("<%s: FORCE OFF>", outputName))
			case force && !toggle:
				outputs = append(outputs, fmt.Sprintf("<%s: FORCE ON>", outputName))
			case !force:
				outputs = append(outputs, fmt.Sprintf("<%s: TOGGLE>", outputName))
			}
		}
	}

	return strings.Join(outputs, ",")
}

func decodeStatus(src, dst int, payload []byte) string {
	if payload[0] != 0x30 || dst != 4 {
		return defaultPayloadParser(src, dst, payload)
	}

	outputs := make([]string, 0)

	for i := 0; i < 8; i++ {
		if testDigit(payload[1], i) {
			outputs = append(outputs, mapOutputIfPossible(src, i))
		}
	}

	return strings.Join(outputs, ",")
}
