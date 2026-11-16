package monitor

import (
	"time"

	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
)

type ExportedMessage struct {
	Packet    lcn.Packet
	LastSeen  time.Time
	Times     int
	Formatted string
}

func ParsePayloadIfPossible(moduleOutputs map[int]map[int]string, src, dst, cmd int, payload []byte) string {
	return parsePayloadIfPossible(moduleOutputs, src, dst, cmd, payload)
}

func CheckDigit(b byte, out int) bool {
	return testDigit(b, out)
}

func MapIDIfPossible(idMap map[int]string, value int) string {
	return mapIfPossible(idMap, value)
}

func MapCmdIfPossible(value int) string {
	return mapIfPossible(cmdMap, value)
}

func MapOutputIfPossible(moduleOutputs map[int]map[int]string, module, output int) string {
	return mapOutputIfPossible(moduleOutputs, module, output)
}

func (d *DataStore) GetLast(n int) []ExportedMessage {
	msgs := d.getLast(n)
	res := make([]ExportedMessage, 0, len(msgs))

	for i := range msgs {
		res = append(res, ExportedMessage{
			Packet:    msgs[i].Packet,
			LastSeen:  msgs[i].lastSeen,
			Times:     msgs[i].times,
			Formatted: d.formatMessage(&msgs[i]),
		})
	}

	return res
}

func (d *DataStore) RenderData() [][]string {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.renderData()
}
