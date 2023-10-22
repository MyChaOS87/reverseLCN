package main

import (
	"fmt"

	"github.com/MyChaOS87/reverseLCN.git/internal/cmd"
	"github.com/MyChaOS87/reverseLCN.git/internal/mqtt"
	"github.com/MyChaOS87/reverseLCN.git/internal/serial"
	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
)

func main() {
	ctx, cancel, cfg := cmd.Init()
	defer cancel()

	publisher := mqtt.NewPublisher(
		mqtt.Broker(cfg.Mqtt.Broker))
	publisher.Run(ctx, cancel)

	reader := serial.NewReader(
		serial.BaudRate(cfg.Serial.BaudRate),
		serial.PortName(cfg.Serial.Port))
	reader.Run(ctx, cancel, func(lcn serial.LcnPacket) {
		log.Infof("%s", lcn.ToNiceString())
		publisher.
			ToTopic(
				fmt.Sprintf("%s/segment/%d/target/%d/",
					cfg.Mqtt.RootTopic,
					lcn.Seg,
					lcn.Dst)).
			Publish(lcn)
	})

	<-ctx.Done()

	log.Errorf("context done: %s", ctx.Err().Error())
}
