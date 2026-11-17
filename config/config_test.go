package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/reverseLCN/config"
	"github.com/MyChaOS87/reverseLCN/pkg/hidden"
)

func TestLoadAndParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		filename        string
		content         string
		useExplicitFile bool
		wantErr         error
		wantBroker      string
		wantIDMap       map[int]string
		wantOutputs     map[int]map[int]string
	}{
		{
			name:     "omitted monitor section",
			filename: "config.yml",
			content: `---
mqtt:
  broker: tcp://mosquitto.internal:1883
  rootTopic: lcn
serial:
  port: /dev/ttyUSB0
  baudRate: 9600
`,
			wantBroker:  "tcp://mosquitto.internal:1883",
			wantIDMap:   nil,
			wantOutputs: nil,
		},
		{
			name:     "populated monitor mappings via prefix path",
			filename: "config.yml",
			content: `---
mqtt:
  broker: tcp://mosquitto.internal:1883
  rootTopic: lcn
monitor:
  idMap:
    4: Display
    11: Switch Kitchen
    33: R8H 33 - Relay A
  moduleOutputs:
    33:
      0: Relay Living Room
`,
			wantBroker: "tcp://mosquitto.internal:1883",
			wantIDMap: map[int]string{
				4:  "Display",
				11: "Switch Kitchen",
				33: "R8H 33 - Relay A",
			},
			wantOutputs: map[int]map[int]string{
				33: {
					0: "Relay Living Room",
				},
			},
		},
		{
			name:            "explicit yml file path",
			filename:        "custom.yml",
			useExplicitFile: true,
			content: `---
mqtt:
  broker: tcp://broker.example:1883
  rootTopic: lcn
monitor:
  idMap:
    4: Display
`,
			wantBroker: "tcp://broker.example:1883",
			wantIDMap: map[int]string{
				4: "Display",
			},
			wantOutputs: nil,
		},
		{
			name:     "missing config file returns ErrConfigFileNotFound",
			filename: "missing.yml",
			wantErr:  config.ErrConfigFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			fullPath := filepath.Join(dir, tt.filename)

			if tt.content != "" {
				require.NoError(t, os.WriteFile(fullPath, []byte(tt.content), 0o600))
			}

			arg := stringsTrimExt(fullPath)
			if tt.useExplicitFile {
				arg = fullPath
			}

			v, err := config.LoadConfig(arg)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, v)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, v)

			cfg, err := config.ParseConfig(v)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			assert.Equal(t, tt.wantBroker, cfg.Mqtt.Broker.Reveal())
			assert.Equal(t, tt.wantIDMap, cfg.Monitor.IDMap)
			assert.Equal(t, tt.wantOutputs, cfg.Monitor.ModuleOutputs)
		})
	}
}

func TestParseConfigNilViper(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseConfig(nil)
	require.ErrorIs(t, err, config.ErrNilViper)
	assert.Nil(t, cfg)
}

func TestLoadAndParseConfigEnvOverridesAndRedaction(t *testing.T) {
	const (
		rawSecret   = "s3cr3t-pass"
		rawBroker   = "tls://svc:inline-secret@mosquitto.internal:8883"
		wantBroker  = "tls://svc:" + hidden.HiddenText + "@mosquitto.internal:8883"
		wantCert    = "/etc/tls/client.crt"
		wantCA      = "/etc/tls/ca.crt"
		wantKey     = "/etc/tls/client.key"
		wantClient  = "lcn2mqtt-pod-0"
		wantUser    = "lcn-bridge"
		wantSerial  = "/dev/lcn-pku"
		wantSNIHost = "mosquitto.internal"
	)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	// Omit clientId, username, password, tls.* from YAML to prove env vars bind even when absent from file.
	content := `---
mqtt:
  broker: tcp://mosquitto.internal:1883
  rootTopic: lcn
serial:
  port: /dev/ttyUSB0
  baudRate: 9600
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	t.Setenv("LCN_SERIAL_PORT", wantSerial)
	t.Setenv("LCN_MQTT_BROKER", rawBroker)
	t.Setenv("LCN_MQTT_CLIENTID", wantClient)
	t.Setenv("LCN_MQTT_USERNAME", wantUser)
	t.Setenv("LCN_MQTT_PASSWORD", rawSecret)
	t.Setenv("LCN_MQTT_TLS_ENABLED", "true")
	t.Setenv("LCN_MQTT_TLS_CAFILE", wantCA)
	t.Setenv("LCN_MQTT_TLS_CERTFILE", wantCert)
	t.Setenv("LCN_MQTT_TLS_KEYFILE", wantKey)
	t.Setenv("LCN_MQTT_TLS_SERVERNAME", wantSNIHost)
	t.Setenv("LCN_MQTT_TLS_INSECURESKIPVERIFY", "true")

	v, err := config.LoadConfig(cfgPath)
	require.NoError(t, err)

	cfg, err := config.ParseConfig(v)
	require.NoError(t, err)

	assert.Equal(t, wantSerial, cfg.Serial.Port)
	assert.Equal(t, rawBroker, cfg.Mqtt.Broker.Reveal())
	assert.Equal(t, wantClient, cfg.Mqtt.ClientID)
	assert.Equal(t, wantUser, cfg.Mqtt.Username)
	assert.Equal(t, rawSecret, cfg.Mqtt.Password.Reveal())
	assert.True(t, cfg.Mqtt.TLS.Enabled)
	assert.Equal(t, wantCA, cfg.Mqtt.TLS.CAFile)
	assert.Equal(t, wantCert, cfg.Mqtt.TLS.CertFile)
	assert.Equal(t, wantKey, cfg.Mqtt.TLS.KeyFile)
	assert.Equal(t, wantSNIHost, cfg.Mqtt.TLS.ServerName)
	assert.True(t, cfg.Mqtt.TLS.InsecureSkipVerify)

	formattedPlusV := fmt.Sprintf("%+v", cfg)
	formattedSharpV := fmt.Sprintf("%#v", cfg)

	assert.NotContains(t, formattedPlusV, rawSecret)
	assert.NotContains(t, formattedPlusV, "inline-secret")
	assert.Contains(t, formattedPlusV, hidden.HiddenText)
	assert.Contains(t, formattedPlusV, wantBroker)

	assert.NotContains(t, formattedSharpV, rawSecret)
	assert.NotContains(t, formattedSharpV, "inline-secret")
	assert.Contains(t, formattedSharpV, hidden.HiddenText)
}

func stringsTrimExt(p string) string {
	ext := filepath.Ext(p)
	if ext == "" {
		return p
	}

	return p[:len(p)-len(ext)]
}
