//nolint:gochecknoglobals
package main

import (
	"fmt"

	"github.com/MyChaOS87/reverseLCN/internal/cmd"
	"github.com/MyChaOS87/reverseLCN/internal/monitor"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/broker"
	"github.com/MyChaOS87/reverseLCN/pkg/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN/pkg/broker/null"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	var broker broker.Broker
	if cfg.Mqtt.Enabled {
		broker = mqtt.NewBroker(
			mqtt.Broker(cfg.Mqtt.Broker))
	} else {
		broker = null.NewBroker()
	}

	dataStore := monitor.NewDataStore()

	broker.Run(ctx, cancel)

	broker.Topic(fmt.Sprintf(
		"%s/#",
		cfg.Mqtt.RootTopic)).
		Subscribe(lcn.LcnPacket{}, func(_ string, in interface{}) {
			if pkt, ok := in.(*lcn.LcnPacket); ok {
				dataStore.Add(*pkt)
			}
		})

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
