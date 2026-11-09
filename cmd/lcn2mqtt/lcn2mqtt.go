package main

import (
	"fmt"

	"github.com/MyChaOS87/reverseLCN/internal/cmd"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	broker := mqtt.NewBroker(
		mqtt.BrokerURL(cfg.Mqtt.Broker))

	port := serial.NewPort(
		serial.BaudRate(cfg.Serial.BaudRate),
		serial.PortName(cfg.Serial.Port),
		serial.Deserializer(lcn.Deserialize),
	)
	port.Run(ctx, cancel, func(pkt packet.Packet) {
		log.Infof("%s", pkt.ToNiceString())

		if lcnPkt, ok := pkt.(*lcn.Packet); ok {
			if err := broker.
				Topic(
					fmt.Sprintf("%s/segment/%d/target/%d/",
						cfg.Mqtt.RootTopic,
						lcnPkt.Seg,
						lcnPkt.Dst)).
				Publish(lcnPkt); err != nil {
				log.Errorf("Could not publish LCN packet: %v", err)
			}
		} else {
			log.Debug("Not a LCN Packet")
		}
	})

	broker.Topic(
		fmt.Sprintf(
			"%s/in",
			cfg.Mqtt.RootTopic)).
		Subscribe(func(_ string, pkt lcn.Packet) {
			log.Infof("MQTT callback got LCN: %s", pkt.ToNiceString())

			buf, err := pkt.Serialize()
			if err != nil {
				log.Errorf("Could not Serialize LCN (%s): %v", pkt.ToNiceString(), err)

				return
			}

			go port.Send(buf)
		})

	if err := broker.Run(ctx); err != nil {
		log.Errorf("mqtt broker stopped with error: %v", err)
	}
}
