//nolint:gochecknoglobals
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/pterm/pterm"

	"github.com/MyChaOS87/reverseLCN.git/internal/cmd"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN.git/pkg/broker"
	"github.com/MyChaOS87/reverseLCN.git/pkg/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN.git/pkg/broker/null"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
)

type ds struct {
	lcn.LcnPacket
	lastSeen time.Time
	times    int
}

var data map[string]ds

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
		2: "Wandlampe Wohnen Aussen",
		3: "Deckenlampe Wohnen West",
		5: "Aussenlicht Ost",
		6: "Aussenlicht Südwest",
		7: "Deckenlampe Süd",
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
	104: "status",
	19:  "switch",
}

var payloadParserByCommand = map[int]func(src, dst int, payload []byte) string{
	19:  decodeSwitch,
	104: decodeStatus,
}

func defaultPayloadParser(_, _ int, payload []byte) string {
	return hex.EncodeToString(payload)
}

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	data = make(map[string]ds)

	var broker broker.Broker
	if cfg.Mqtt.Enabled {
		broker = mqtt.NewBroker(
			mqtt.Broker(cfg.Mqtt.Broker))
	} else {
		broker = null.NewBroker()
	}

	broker.Run(ctx, cancel)
	ui(ctx, cancel)

	broker.Topic(fmt.Sprintf(
		"%s/#",
		cfg.Mqtt.RootTopic)).
		Subscribe(lcn.LcnPacket{}, func(_ string, in interface{}) {
			if pkt, ok := in.(*lcn.LcnPacket); ok {
				id := pkt.ToString()
				if v, ok := data[id]; ok {
					v.times++
					v.lastSeen = time.Now()
					data[id] = v
				} else {
					data[id] = ds{
						LcnPacket: *pkt,
						times:     1,
						lastSeen:  time.Now(),
					}
				}
			}
		})

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}

func ui(ctx context.Context, cancel context.CancelFunc) {
	header := pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgRed))

	pterm.DefaultCenter.Printfln(header.Sprintf("LCN Monitor"))

	area, _ := pterm.DefaultArea.WithFullscreen(true).Start()

	go func() {
		const tickerInterval = 500 * time.Millisecond

		ticker := time.NewTicker(tickerInterval)

		for {
			select {
			case <-ticker.C:
				table, _ := pterm.DefaultTable.WithData(renderData()).WithHasHeader(true).Srender()
				area.Update(pterm.DefaultCenter.Sprint(table))
			case <-ctx.Done():
				cancel()

				return
			}
		}
	}()
}

func renderData() [][]string {
	dataSlice := make([]ds, 0, len(data))

	for _, v := range data {
		dataSlice = append(dataSlice, v)
	}

	slices.SortFunc(dataSlice, func(a, b ds) int {
		return -a.lastSeen.Compare(b.lastSeen)
	})

	lines := make([][]string, 0, len(dataSlice))

	lineLength := 7
	for _, v := range dataSlice {
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

	return fmt.Sprintf("<unnamed> %d", output)
}

func parsePayloadIfPossible(src, dst, cmd int, payload []byte) string {
	if f, ok := payloadParserByCommand[cmd]; ok {
		return f(src, dst, payload)
	}

	return defaultPayloadParser(src, dst, payload)
}

func decodeSwitch(src, dst int, payload []byte) string {
	if m, ok := moduleOutputs[dst]; ok {
		result := parseOutputOnModule(payload, m, dst)

		if result != "" {
			return fmt.Sprintf("%s %s", hex.EncodeToString(payload[0:1]), result)
		}
	}

	return defaultPayloadParser(src, dst, payload)
}

func decodeStatus(src, dst int, payload []byte) string {
	if payload[0] != 0x30 || dst != 4 {
		return defaultPayloadParser(src, dst, payload)
	}

	if m, ok := moduleOutputs[src]; ok {
		result := parseOutputOnModule(payload, m, src)

		if result != "" {
			return result
		}
	}

	return defaultPayloadParser(src, dst, payload)
}

func parseOutputOnModule(payload []byte, module map[int]string, moduleId int) string {
	var result string

	for out := 0; out < 8; out++ {
		if payload[1]&(1<<uint(out)) == 0 {
			continue
		}

		if o, ok := module[out]; ok {
			result = fmt.Sprintf("%s<%s>", result, o)
		} else {
			result = fmt.Sprintf("%s<UNKNOWN %d-%d>", result, moduleId, out)
		}
	}
	return result
}
