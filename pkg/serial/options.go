package serial

import (
	"time"

	"go.bug.st/serial"

	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/packet"
	"github.com/MyChaOS87/reverseLCN/pkg/serial/chunker/plain"
)

const (
	defaultBaudRate      = 9600
	defaultDataBits      = 8
	defaultSendQueueSize = 128
	defaultReadTimeout   = 100 * time.Millisecond
)

// PortOpenerFunc opens a serial device with the given mode.
type PortOpenerFunc func(portName string, mode *serial.Mode) (serial.Port, error)

type (
	Option func(*Config)
	Config struct {
		portName      string
		baudRate      int
		parity        serial.Parity
		dataBits      int
		stopBits      serial.StopBits
		deserializer  packet.Deserializer
		minLength     int
		sendQueueSize int
		readTimeout   time.Duration
		openPort      PortOpenerFunc
	}
)

func PortName(portName string) Option {
	return func(c *Config) {
		c.portName = portName
	}
}

func BaudRate(baudRate int) Option {
	return func(c *Config) {
		c.baudRate = baudRate
	}
}

func Parity(parity serial.Parity) Option {
	return func(c *Config) {
		c.parity = parity
	}
}

func DataBits(dataBits int) Option {
	return func(c *Config) {
		c.dataBits = dataBits
	}
}

func StopBits(stopBits serial.StopBits) Option {
	return func(c *Config) {
		c.stopBits = stopBits
	}
}

func Deserializer(deserializer packet.Deserializer) Option {
	return func(c *Config) {
		c.deserializer = deserializer
	}
}

func MinLength(minLength int) Option {
	return func(c *Config) {
		c.minLength = minLength
	}
}

// SendQueueSize configures the outbound serial frame queue capacity.
func SendQueueSize(size int) Option {
	return func(c *Config) {
		if size > 0 {
			c.sendQueueSize = size
		}
	}
}

// ReadTimeout configures the inter-byte read timeout used to detect an idle bus.
func ReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		if timeout > 0 {
			c.readTimeout = timeout
		}
	}
}

// PortOpener overrides the function used to open the underlying serial device.
func PortOpener(opener PortOpenerFunc) Option {
	return func(c *Config) {
		if opener != nil {
			c.openPort = opener
		}
	}
}

func newDefaultConfig() *Config {
	return &Config{
		portName:      "/dev/ttyACM0",
		baudRate:      defaultBaudRate,
		parity:        serial.NoParity,
		dataBits:      defaultDataBits,
		stopBits:      serial.OneStopBit,
		deserializer:  plain.Deserialize,
		minLength:     1,
		sendQueueSize: defaultSendQueueSize,
		readTimeout:   defaultReadTimeout,
		openPort:      serial.Open,
	}
}
