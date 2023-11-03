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
	} else {
		publisher = null.NewPublisher()
	}

	port := serial.NewPort(
		serial.BaudRate(cfg.Serial.BaudRate),
		serial.PortName(cfg.Serial.Port),
		serial.Deserializer(lcn.Deserialize),
	)
	port.Run(ctx, cancel, func(pkt packet.Packet) {
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

	publisher.Topic("lcnIn").Subscribe(lcn.LcnPacket{}, func(data interface{}) {
		if pkt, ok := data.(*lcn.LcnPacket); ok {
			log.Infof("MQTT callback got LCN: %s", pkt.ToNiceString())
			buf, err := pkt.Serialize()
			if err != nil {
				log.Error("Could not Serialize LCN: %s", pkt.ToNiceString())
				return
			}

			go port.Send(buf)
		} else {
			log.Errorf("Could not interpret MQTT: %s", data)
		}
	})

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
