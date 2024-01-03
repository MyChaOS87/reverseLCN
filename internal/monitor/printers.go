package monitor

import (
	"fmt"
	"slices"
)

//nolint:deadcode,unused
func renderData(data map[string]*message) [][]string {
	dataSlice := make([]*message, 0, len(data))

	for _, v := range data {
		dataSlice = append(dataSlice, v)
	}

	slices.SortFunc(dataSlice, func(a, b *message) int {
		return -a.lastSeen.Compare(b.lastSeen)
	})

	lines := make([][]string, 0, len(dataSlice))

	for _, v := range dataSlice {
		const lineLength = 7

		line := make([]string, 0, lineLength)
		line = append(line, v.lastSeen.Local().Format("2006-01-02 15:04:05 MST"))
		line = append(line, fmt.Sprintf("%d", v.times))
		line = append(line, mapIfPossible(idMap, int(v.Src)))
		line = append(line, fmt.Sprintf("%d", v.Seg))
		line = append(line, mapIfPossible(idMap, int(v.Dst)))
		line = append(line, mapIfPossible(cmdMap, int(v.Cmd)))
		line = append(line, parsePayloadIfPossible(int(v.Src), int(v.Dst), int(v.Cmd), v.Payload))

		lines = append(lines, line)
	}

	result := make([][]string, 0, len(lines)+1)
	result = append(result, []string{"last seen", "#", "Src", "Seg", "Dst", "Command", "Payload"})
	result = append(result, lines...)

	return result
}
