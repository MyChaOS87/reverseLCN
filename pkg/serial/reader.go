package serial

import (
	"context"
	"time"

	"go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker"
)

const bufferSize = 1024

type Reader interface {
	Run(ctx context.Context, cancel context.CancelFunc, eject chunker.EjectFunc)
}

type reader struct {
	portName string
	mode     serial.Mode
	chunker  chunker.Chunker
}

func (r *reader) Run(ctx context.Context, cancel context.CancelFunc, eject chunker.EjectFunc) {
	port, err := serial.Open(r.portName, &r.mode)
	if err != nil {
		log.Errorf("Cannot Open Port %s: %s", r.portName, err.Error())
		cancel()
		return
	}

	// ensure that we try to read twice per bufferSize using the baudRate, thus we should never read to slow to catch everything given enough resources
	// ticker := time.NewTicker((bufferSize * time.Second) / time.Duration(r.mode.BaudRate*2))
	ticker := time.NewTicker(8 * time.Millisecond)

	go func() {
		defer port.Close()
		defer ticker.Stop()

		for {
			buffer := make([]byte, bufferSize)

			select {
			case <-ticker.C:
				len, err := port.Read(buffer)
				if err != nil {
					log.Errorf("Error reading from serial(%s): %s", r.portName, err.Error())
					cancel()
					return
				}

				r.chunker.Collect(buffer[0:len], eject)
			case <-ctx.Done():
				log.Errorf("Context done: %s", ctx.Err())
				return
			}
		}
	}()
}

func NewReader(options ...Option) Reader {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	return &reader{
		portName: config.portName,
		mode: serial.Mode{
			BaudRate: config.baudRate,
			Parity:   config.parity,
			DataBits: config.dataBits,
			StopBits: config.stopBits,
		},
		chunker: chunker.NewChunker(config.deserializer, config.minLength),
	}
}
