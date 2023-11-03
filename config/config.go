package config

import (
	"log"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	loggerConfig "github.com/MyChaOS87/reverseLCN.git/pkg/log/config"
)

const (
	envPrefix = "LCN"
)

type SerialConfig struct {
	Port     string
	BaudRate int
}

type MqttConfig struct {
	Broker    string
	RootTopic string
	Enabled   bool
}

// Config struct.
type Config struct {
	Logger loggerConfig.Logger
	Serial SerialConfig
	Mqtt   MqttConfig
}

// LoadConfig loads config file from given path.
func LoadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigName(filename)
	v.AddConfigPath(".")
	v.AddConfigPath("./")
	v.AddConfigPath("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if ok := errors.As(err, &viper.ConfigFileNotFoundError{}); ok {
			return nil, errors.New("config file not found")
		}

		return nil, errors.Wrap(err, "failed to read config")
	}

	return v, nil
}

// ParseConfig parses config file.
func ParseConfig(v *viper.Viper) (*Config, error) {
	var c Config

	if err := v.Unmarshal(&c); err != nil {
		log.Printf("unable to decode into struct, %v", err)

		return nil, errors.Wrap(err, "failed to parse config")
	}

	return &c, nil
}
