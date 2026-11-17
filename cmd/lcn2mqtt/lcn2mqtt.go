package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/MyChaOS87/reverseLCN/internal/cmd"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/broker/mqtt"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
)

// Version is stamped at build time via -ldflags.
var Version = "dev" //nolint:gochecknoglobals // build-time ldflags version

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "lcn2mqtt: %v\n", err)

		os.Exit(1)
	}
}

func run() error {
	ctx, cancel, cfg, err := cmd.Init()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	defer cancel()

	log.Infof("Starting lcn2mqtt version %s", Version)

	broker, err := cmd.NewMQTTBroker(cfg.Mqtt)
	if err != nil {
		return fmt.Errorf("configure mqtt broker: %w", err)
	}

	port := serial.NewPort(
		serial.BaudRate(cfg.Serial.BaudRate),
		serial.PortName(cfg.Serial.Port),
		serial.Deserializer(lcn.Deserialize),
		serial.MinLength(lcn.MinPacketLength),
	)

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	broker.Topic(fmt.Sprintf("%s/in", cfg.Mqtt.RootTopic)).
		Subscribe(func(_ string, pkt lcn.Packet) {
			log.Infof("MQTT callback got LCN: %s", pkt.ToNiceString())

			buf, serializeErr := pkt.Serialize()
			if serializeErr != nil {
				log.Errorf("Could not Serialize LCN (%s): %v", pkt.ToNiceString(), serializeErr)

				return
			}

			if sendErr := port.Send(runCtx, buf); sendErr != nil {
				log.Errorf("Could not enqueue LCN frame to serial port: %v", sendErr)
			}
		})

	return runBridge(runCtx, runCancel, port, broker, publishEjectedPacket(cfg.Mqtt.RootTopic, broker))
}

func runBridge(
	ctx context.Context,
	cancel context.CancelFunc,
	port serial.Port,
	broker *mqtt.Broker,
	eject func(packet.Packet),
) error {
	errCh := make(chan error, 2)

	var wg sync.WaitGroup

	wg.Go(func() {
		defer cancel()

		if portErr := port.Run(ctx, eject); portErr != nil {
			errCh <- fmt.Errorf("serial port stopped: %w", portErr)
		}
	})

	wg.Go(func() {
		defer cancel()

		if brokerErr := broker.Run(ctx); brokerErr != nil {
			errCh <- fmt.Errorf("mqtt broker stopped: %w", brokerErr)
		}
	})

	wg.Wait()
	close(errCh)

	var runErrs []error
	for e := range errCh {
		runErrs = append(runErrs, e)
	}

	if err := errors.Join(runErrs...); err != nil {
		log.Errorf("bridge exited with error: %v", err)

		return err
	}

	return nil
}

func publishEjectedPacket(rootTopic string, broker *mqtt.Broker) func(packet.Packet) {
	return func(pkt packet.Packet) {
		log.Infof("%s", pkt.ToNiceString())

		if lcnPkt, ok := pkt.(*lcn.Packet); ok {
			topic := fmt.Sprintf("%s/segment/%d/target/%d/", rootTopic, lcnPkt.Seg, lcnPkt.Dst)
			if err := broker.Topic(topic).Publish(lcnPkt); err != nil {
				log.Errorf("Could not publish LCN packet: %v", err)
			}
		} else {
			log.Debug("Not a LCN Packet")
		}
	}
}
