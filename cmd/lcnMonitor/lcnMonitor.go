package main

import (
	"fmt"
	"os"

	"github.com/MyChaOS87/reverseLCN/internal/cmd"
	"github.com/MyChaOS87/reverseLCN/internal/monitor"
	"github.com/MyChaOS87/reverseLCN/internal/serial/chunker/lcn"
	"github.com/MyChaOS87/reverseLCN/pkg/log"
)

// Version is stamped at build time via -ldflags.
var Version = "dev" //nolint:gochecknoglobals // build-time ldflags version

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "lcnMonitor: %v\n", err)

		os.Exit(1)
	}
}

func run() error {
	ctx, cancel, cfg, err := cmd.Init()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	defer cancel()

	log.Infof("Starting lcnMonitor version %s", Version)

	broker, err := cmd.NewMQTTBroker(cfg.Mqtt)
	if err != nil {
		return fmt.Errorf("configure mqtt broker: %w", err)
	}

	dataStore := monitor.NewDataStore(
		monitor.WithIDMap(cfg.Monitor.IDMap),
		monitor.WithModuleOutputs(cfg.Monitor.ModuleOutputs),
	)

	broker.Topic(fmt.Sprintf(
		"%s/#",
		cfg.Mqtt.RootTopic)).
		Subscribe(func(_ string, pkt lcn.Packet) {
			dataStore.Add(pkt)
		})

	if err := broker.Run(ctx); err != nil {
		log.Errorf("mqtt broker stopped with error: %v", err)

		return fmt.Errorf("mqtt broker stopped: %w", err)
	}

	return nil
}
