package serial

import "go.bug.st/serial"

type (
	Option func(*Config)
	Config struct {
		portName string
		baudRate int
		parity   serial.Parity
		dataBits int
		stopBits serial.StopBits
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

func newDefaultConfig() *Config {
	return &Config{
		portName: "/dev/ttyACM0",
		baudRate: 9600,
		parity:   serial.NoParity,
		dataBits: 8,
		stopBits: serial.OneStopBit,
	}
}
