package main

import (
	"fmt"

	"github.com/MyChaOS87/reverseLCN.git/internal/broker"
	"github.com/MyChaOS87/reverseLCN.git/internal/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN.git/internal/broker/null"
	"github.com/MyChaOS87/reverseLCN.git/internal/cmd"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	var publisher broker.Publisher
	if cfg.Mqtt.Enabled {
		publisher = mqtt.NewPublisher(
			mqtt.Broker(cfg.Mqtt.Broker))
		publisher.Run(ctx, cancel)

		publisher.Topic("lcnIn").Subscribe(lcn.LcnPacket{}, func(data interface{}) {
			if pkt, ok := data.(*lcn.LcnPacket); ok {
				log.Infof("MQTT callback got LCN: %s", pkt.ToNiceString())
			} else {
				log.Errorf("Could not interpret MQTT: %s", data)
			}
		})
	} else {
		publisher = null.NewPublisher()
	}

	reader := serial.NewReader(
		serial.BaudRate(cfg.Serial.BaudRate),
		serial.PortName(cfg.Serial.Port),
		serial.Deserializer(lcn.Deserialize),
	)
	reader.Run(ctx, cancel, func(pkt packet.Packet) {
		log.Infof("%s", pkt.ToNiceString())

		if lcn, ok := pkt.(*lcn.LcnPacket); ok {
			publisher.
				Topic(
					fmt.Sprintf("%s/segment/%d/target/%d/",
						cfg.Mqtt.RootTopic,
						lcn.Seg,
						lcn.Dst)).
				Publish(lcn)
		} else {
			log.Debug("Not a LCN Packet")
		}
	})

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
