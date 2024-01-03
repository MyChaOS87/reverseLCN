package monitor

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MyChaOS87/automqttion.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/lcn"
)

type DataStore struct {
	messages map[string]*message
	mutex    sync.Mutex
}

type message struct {
	lcn.LcnPacket
	lastSeen time.Time
	times    int
}

func (d *DataStore) Add(pkt lcn.LcnPacket) {
	now := time.Now()

	d.mutex.Lock()
	defer d.mutex.Unlock()
	{

		key := pkt.ToString()
		if v, ok := d.messages[key]; ok {
			v.Update(now)

			log.Infof("UPD: %s", v.ToString())
		} else {
			m := message{
				LcnPacket: pkt,
				lastSeen:  now,
				times:     1,
			}
			d.messages[key] = &m

			log.Infof("ADD: %s", m.ToString())
		}
	}
}

func (d *DataStore) GetLast(n int) []message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	result := make([]message, 0, len(d.messages))
	for _, v := range d.messages {
		result = append(result, *v)
	}

	slices.SortFunc(result, func(a, b message) int {
		return -a.lastSeen.Compare(b.lastSeen)
	})

	return result[:n]
}

func (m *message) Update(lastSeen time.Time) {
	m.lastSeen = lastSeen
	m.times++
}

func (m *message) ToString() string {
	const lineLength = 7

	line := make([]string, 0, lineLength)
	line = append(line, m.lastSeen.Local().Format("2006-01-02 15:04:05 MST"))
	line = append(line, fmt.Sprintf("%d", m.times))
	line = append(line, mapIfPossible(idMap, int(m.Src)))
	line = append(line, fmt.Sprintf("%d", m.Seg))
	line = append(line, mapIfPossible(idMap, int(m.Dst)))
	line = append(line, mapIfPossible(cmdMap, int(m.Cmd)))
	line = append(line, parsePayloadIfPossible(int(m.Src), int(m.Dst), int(m.Cmd), m.Payload))

	return strings.Join(line, "\t")
}

func NewDataStore() *DataStore {
	return &DataStore{
		messages: make(map[string]*message),
	}
}
