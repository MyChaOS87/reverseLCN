package main

import (
	"fmt"

	"github.com/MyChaOS87/reverseLCN/internal/cmd"
	"github.com/MyChaOS87/reverseLCN/internal/monitor"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	broker := mqtt.NewBroker(
		mqtt.BrokerURL(cfg.Mqtt.Broker))

	dataStore := monitor.NewDataStore()

	broker.Topic(fmt.Sprintf(
		"%s/#",
		cfg.Mqtt.RootTopic)).
		Subscribe(func(_ string, pkt lcn.Packet) {
			dataStore.Add(pkt)
		})

	if err := broker.Run(ctx); err != nil {
		log.Errorf("mqtt broker stopped with error: %v", err)
	}
}
