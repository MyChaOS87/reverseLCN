package main

import (
	"cmp"
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/MyChaOS87/reverseLCN.git/internal/broker"
	"github.com/MyChaOS87/reverseLCN.git/internal/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN.git/internal/broker/null"
	"github.com/MyChaOS87/reverseLCN.git/internal/cmd"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/pterm/pterm"
)

type ds struct {
	lcn.LcnPacket
	times int
}

var data map[string]ds

var dst_map = map[int]string{
	4: "Display",
}

var cmd_map = map[int]string{
	104: "status",
	19:  "switch",
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
					data[id] = v
				} else {
					data[id] = ds{
						LcnPacket: *pkt,
						times:     1,
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
		ticker := time.NewTicker(500 * time.Millisecond)
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
	lines := make([][]string, 0, len(data))

	lineLength := 6
	for _, v := range data {
		line := make([]string, 0, lineLength)
		line = append(line, fmt.Sprintf("%d", v.times))
		line = append(line, fmt.Sprintf("%d", v.Src))
		line = append(line, fmt.Sprintf("%d", v.Seg))
		line = append(line, mapIfPossible(dst_map, int(v.Dst)))
		line = append(line, mapIfPossible(cmd_map, int(v.Cmd)))
		line = append(line, hex.EncodeToString(v.Payload))

		lines = append(lines, line)
	}
	slices.SortFunc(lines, func(a, b []string) int {
		result := 0
		for pos := 0; result == 0 && pos < lineLength; pos++ {
			result = cmp.Compare(a[pos], b[pos])
		}
		return -result
	})

	result := make([][]string, 0, len(lines)+1)
	result = append(result, []string{"#", "Src", "Seg", "Dst", "Command", "Payload"})
	result = append(result, lines...)

	return result
}

func mapIfPossible(m map[int]string, value int) string {
	if s, ok := m[value]; ok {
		return s
	}

	return fmt.Sprintf("%d", value)
}
