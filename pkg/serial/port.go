package serial

import (
	"context"
	"time"

	"go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker"
)

const (
	bufferSize           = 1024
	defaultSendQueueSize = 10
)

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

	//nolint:mnd
	err = port.SetReadTimeout(100 * time.Millisecond)
	if err != nil {
		log.Errorf("Cannot set read timeout on serial(%s): %s", p.portName, err.Error())
		port.Close()
		cancel()

		return
	}

	// Close port immediately when context is cancelled to unblock any pending reads
	go func() {
		<-ctx.Done()
		port.Close()
	}()

	go p.startWriter(ctx, port)
	go p.startReader(ctx, cancel, port, eject)
}

func (p *port) startWriter(ctx context.Context, port serial.Port) {
	for {
		select {
		case message := <-p.sendQueue:
			length, err := port.Write(message)
			switch {
			case err != nil:
				log.Errorf("Error writing %v to serial(%s): %s", message, p.portName, err.Error())
			case length != len(message):
				log.Errorf("Incomplete write of %v to serial(%s): sent %d", message, p.portName, length)
			default:
				log.Debugf("Wrote %v to serial(%s): ", message, p.portName)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *port) startReader(ctx context.Context, cancel context.CancelFunc, port serial.Port, eject chunker.EjectFunc) {
	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := port.Read(buffer)
			if err != nil {
				// Only log and cancel if the context isn't already done
				if ctx.Err() == nil {
					log.Errorf("Error reading from serial(%s): %s", p.portName, err.Error())
					cancel()
				}

				return
			}

			if n > 0 {
				p.chunker.Collect(buffer[:n], eject)
			}
		}
	}
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

		sendQueue: make(chan []byte, defaultSendQueueSize),
	}
}
