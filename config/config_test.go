package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/reverseLCN/config"
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

			assert.Equal(t, tt.wantBroker, cfg.Mqtt.Broker)
			assert.Equal(t, tt.wantIDMap, cfg.Monitor.IDMap)
			assert.Equal(t, tt.wantOutputs, cfg.Monitor.ModuleOutputs)
		})
	}
}

func stringsTrimExt(p string) string {
	ext := filepath.Ext(p)
	if ext == "" {
		return p
	}

	return p[:len(p)-len(ext)]
}
