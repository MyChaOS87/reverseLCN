package serial

import (
	"context"
	"time"

	"go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN.git/pkg/log"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker"
)

const bufferSize = 1024

type Port interface {
	Run(ctx context.Context, cancel context.CancelFunc, eject chunker.EjectFunc)
	Send(buf []byte)
}

type port struct {
	sendQueue chan []byte

	portName string
	mode     serial.Mode
	chunker  chunker.Chunker
}

func (p *port) Send(buf []byte) {
	p.sendQueue <- buf
}

func (p *port) Run(ctx context.Context, cancel context.CancelFunc, eject chunker.EjectFunc) {
	port, err := serial.Open(p.portName, &p.mode)
	if err != nil {
		log.Errorf("Cannot Open Port %s: %s", p.portName, err.Error())
		cancel()
		return
	}

	// ensure that we try to read twice per bufferSize using the baudRate, thus we should never read to slow to catch everything given enough resources
	// ticker := time.NewTicker((bufferSize * time.Second) / time.Duration(r.mode.BaudRate*2))
	ticker := time.NewTicker(8 * time.Millisecond)

	err = port.SetReadTimeout(100 * time.Millisecond)
	if err != nil {
		log.Errorf("Cannot set read timeout on serial(%s): %s", p.portName, err.Error())
		cancel()
		return
	}

	go func() {
		defer port.Close()
		defer ticker.Stop()

		for {
			select {
			case message := <-p.sendQueue:
				length, err := port.Write(message)
				if err != nil {
					log.Errorf("Error writing %v to serial(%s): %s", message, p.portName, err.Error())
				} else if length != len(message) {
					log.Errorf("Incomplete write of %v to serial(%s): sent %d", message, p.portName, length)
				} else {
					log.Debugf("Wrote %v to serial(%s): ", message, p.portName)
				}
			case <-ticker.C:
				buffer := make([]byte, bufferSize)

				len, err := port.Read(buffer)
				if err != nil {
					log.Errorf("Error reading from serial(%s): %s", p.portName, err.Error())
					cancel()
					return
				}

				p.chunker.Collect(buffer[0:len], eject)
			case <-ctx.Done():
				log.Errorf("Context done: %s", ctx.Err())
				return
			}
		}
	}()
}

func NewPort(options ...Option) Port {
	config := newDefaultConfig()

	for _, opt := range options {
		opt(config)
	}

	return &port{
		portName: config.portName,
		mode: serial.Mode{
			BaudRate: config.baudRate,
			Parity:   config.parity,
			DataBits: config.dataBits,
			StopBits: config.stopBits,
		},
		chunker: chunker.NewChunker(config.deserializer, config.minLength),

		sendQueue: make(chan []byte, 10),
	}
}
