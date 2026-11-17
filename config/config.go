package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/MyChaOS87/reverseLCN/pkg/hidden"
	loggerConfig "github.com/MyChaOS87/reverseLCN/pkg/log/config"
)

const (
	envPrefix = "LCN"
)

var (
	ErrConfigFileNotFound = errors.New("config file not found")
	ErrNilViper           = errors.New("config viper instance is nil")
)

type SerialConfig struct {
	Port     string
	BaudRate int
}

// MqttTLSConfig configures TLS verification and optional client certificates for MQTT connections.
type MqttTLSConfig struct {
	Enabled            bool
	CAFile             string
	CertFile           string
	KeyFile            string
	ServerName         string
	InsecureSkipVerify bool
}

type MqttConfig struct {
	Broker    hidden.URL
	RootTopic string
	ClientID  string
	Username  string
	Password  hidden.String
	TLS       MqttTLSConfig
}

// MonitorConfig holds optional human-readable module and output labels.
type MonitorConfig struct {
	IDMap         map[int]string
	ModuleOutputs map[int]map[int]string
}

// Config struct.
type Config struct {
	Logger  loggerConfig.Logger
	Serial  SerialConfig
	Mqtt    MqttConfig
	Monitor MonitorConfig
}

// LoadConfig loads config file from given path.
func LoadConfig(filename string) (*viper.Viper, error) {
	v := viper.NewWithOptions(viper.ExperimentalBindStruct())

	if filepath.Ext(filename) != "" {
		v.SetConfigFile(filename)
	} else {
		v.SetConfigName(filepath.Base(filename))
		v.AddConfigPath(filepath.Dir(filename))
		v.AddConfigPath(".")
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok || errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigFileNotFound
		}

		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return v, nil
}

// ParseConfig parses config file.
func ParseConfig(v *viper.Viper) (*Config, error) {
	if v == nil {
		return nil, ErrNilViper
	}

	var c Config

	if err := v.Unmarshal(&c); err != nil {
		log.Printf("unable to decode into struct, %v", err)

		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &c, nil
}
