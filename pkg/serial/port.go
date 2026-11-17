package serial

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN/pkg/log"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker"
)

const bufferSize = 1024

var (
	// ErrPortOpen indicates the serial device could not be opened.
	ErrPortOpen = errors.New("open serial port")
	// ErrPortConfig indicates serial port configuration (such as read timeout) failed.
	ErrPortConfig = errors.New("configure serial port")
	// ErrPortRead indicates an unexpected failure while reading from the serial device.
	ErrPortRead = errors.New("read from serial port")
	// ErrSendQueueFull indicates the outbound FIFO queue is full and cannot accept more frames.
	ErrSendQueueFull = errors.New("serial send queue full")
)

type Port interface {
	Run(ctx context.Context, eject chunker.EjectFunc) error
	Send(ctx context.Context, buf []byte) error
}

type port struct {
	sendQueue   chan []byte
	portName    string
	mode        serial.Mode
	chunker     chunker.Chunker
	readTimeout time.Duration
	openPort    PortOpenerFunc
}

// Send enqueues a copy of buf in strict FIFO order without blocking callers.
func (p *port) Send(ctx context.Context, buf []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("serial send cancelled: %w", err)
	}

	frame := slices.Clone(buf)

	select {
	case <-ctx.Done():
		return fmt.Errorf("serial send cancelled: %w", ctx.Err())
	case p.sendQueue <- frame:
		return nil
	default:
		return ErrSendQueueFull
	}
}

// Run opens the serial port, processes RX/TX loops until ctx cancels or a fatal error occurs,
// and joins all internal goroutines before returning.
func (p *port) Run(ctx context.Context, eject chunker.EjectFunc) error {
	dev, err := p.openPort(p.portName, &p.mode)
	if err != nil {
		return fmt.Errorf("%w (%s): %w", ErrPortOpen, p.portName, err)
	}

	err = dev.SetReadTimeout(p.readTimeout)
	if err != nil {
		_ = dev.Close()

		return fmt.Errorf("%w (%s): %w", ErrPortConfig, p.portName, err)
	}

	portCtx, portCancel := context.WithCancel(ctx)
	defer portCancel()

	var (
		wg      sync.WaitGroup
		readErr error
	)

	// Close port immediately when context is cancelled to unblock any pending Read call.
	wg.Go(func() {
		<-portCtx.Done()

		_ = dev.Close()
	})

	wg.Go(func() {
		p.startWriter(portCtx, dev)
	})

	wg.Go(func() {
		defer portCancel()

		readErr = p.startReader(portCtx, dev, eject)
	})

	wg.Wait()

	return readErr
}

func (p *port) startWriter(ctx context.Context, dev serial.Port) {
	for {
		select {
		case message := <-p.sendQueue:
			length, err := dev.Write(message)
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

func (p *port) startReader(ctx context.Context, dev serial.Port, eject chunker.EjectFunc) error {
	buffer := make([]byte, bufferSize)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := dev.Read(buffer)
			if err != nil {
				// Only return an error if the context isn't already done.
				if ctx.Err() == nil {
					return fmt.Errorf("%w (%s): %w", ErrPortRead, p.portName, err)
				}

				return nil
			}

			if n > 0 {
				p.chunker.Collect(buffer[:n], eject)
			} else {
				p.chunker.Flush(eject)
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
		chunker:     chunker.NewChunker(config.deserializer, config.minLength),
		sendQueue:   make(chan []byte, config.sendQueueSize),
		readTimeout: config.readTimeout,
		openPort:    config.openPort,
	}
}
