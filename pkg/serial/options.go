package serial

import (
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/packet"
	"github.com/MyChaOS87/reverseLCN.git/pkg/serial/chunker/plain"
	"go.bug.st/serial"
)

type (
	Option func(*Config)
	Config struct {
		portName     string
		baudRate     int
		parity       serial.Parity
		dataBits     int
		stopBits     serial.StopBits
		deserializer packet.Deserializer
		minLength    int
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

func newDefaultConfig() *Config {
	return &Config{
		portName:     "/dev/ttyACM0",
		baudRate:     9600,
		parity:       serial.NoParity,
		dataBits:     8,
		stopBits:     serial.OneStopBit,
		deserializer: plain.Deserialize,
		minLength:    1,
	}
}
