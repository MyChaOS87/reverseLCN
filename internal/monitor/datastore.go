package monitor

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
)

type DataStore struct {
	messages      map[string]*message
	idMap         map[int]string
	moduleOutputs map[int]map[int]string
	mutex         sync.Mutex
}

type message struct {
	lcn.Packet

	lastSeen time.Time
	times    int
}

func (d *DataStore) Add(pkt lcn.Packet) {
	now := time.Now()

	d.mutex.Lock()
	defer d.mutex.Unlock()

	key := pkt.ToString()
	if v, ok := d.messages[key]; ok {
		v.update(now)

		log.Infof("UPD: %s", d.formatMessage(v))
	} else {
		m := message{
			Packet:   pkt,
			lastSeen: now,
			times:    1,
		}
		d.messages[key] = &m

		log.Infof("ADD: %s", d.formatMessage(&m))
	}
}

func (d *DataStore) getLast(n int) []message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	result := make([]message, 0, len(d.messages))
	for _, v := range d.messages {
		result = append(result, *v)
	}

	slices.SortFunc(result, func(a, b message) int {
		return -a.lastSeen.Compare(b.lastSeen)
	})

	return result[:min(n, len(result))]
}

func (m *message) update(lastSeen time.Time) {
	m.lastSeen = lastSeen
	m.times++
}

func (d *DataStore) formatMessage(m *message) string {
	const lineLength = 7

	line := make([]string, 0, lineLength)
	line = append(line, m.lastSeen.Local().Format("2006-01-02 15:04:05 MST"))
	line = append(line, fmt.Sprintf("%d", m.times))
	line = append(line, mapIfPossible(d.idMap, int(m.Src)))
	line = append(line, fmt.Sprintf("%d", m.Seg))
	line = append(line, mapIfPossible(d.idMap, int(m.Dst)))
	line = append(line, mapIfPossible(cmdMap, int(m.Cmd)))
	line = append(line, parsePayloadIfPossible(d.moduleOutputs, int(m.Src), int(m.Dst), int(m.Cmd), m.Payload))

	return strings.Join(line, "\t")
}

func NewDataStore(opts ...Option) *DataStore {
	ds := &DataStore{
		messages: make(map[string]*message),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(ds)
		}
	}

	return ds
}
