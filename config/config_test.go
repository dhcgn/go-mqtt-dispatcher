package config

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                 string
		mockReadFile         func(path string) ([]byte, error)
		expectedError        bool
		expectedErrorMessage string
		expectedConfig       *RootConfig
	}{
		{
			name: "ValidConfig",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - operation: "sum"
    color-script: | 
      function get_color(v) {return "#FF00FF";}
`), nil
			},
			expectedError: false,
			expectedConfig: &RootConfig{
				Mqtt: MqttConfig{
					BrokerAsUri: &url.URL{Scheme: "tcp", Host: "localhost:1883"},
				},
				DispatcherEntries: []Entry{
					{
						Operation:           string(OperatorSum),
						ColorScriptCallback: func(float64) (string, error) { return "#FF00FF", nil },
					},
				},
			},
		},
		{
			name: "FileNotFound",
			mockReadFile: func(path string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			expectedError:        true,
			expectedErrorMessage: "file does not exist",
			expectedConfig:       nil,
		},
		{
			name: "InvalidYaml",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`invalid_yaml`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "yaml: unmarshal errors",
			expectedConfig:       nil,
		},
		{
			name: "InvalidBrokerUrl",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "://invalid_url"
dispatcher-entries:
  - operation: "none"
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "parse \"://invalid_url\": missing protocol scheme",
			expectedConfig:       nil,
		},
		{
			name: "InvalidOperation",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - operation: "invalid"
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "ERROR: INVALID OPERATION INDEX 0: 'invalid'",
			expectedConfig:       nil,
		},
		{
			name: "ColorScriptError",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - operation: "sum"
    color-script: | 
      return 'red';
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "ERROR RUNNING SCRIPT",
			expectedConfig:       nil,
		},
		{
			name: "InvalidFallbackMode",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - fallback:
      mode: "bogus"
      after: "1h"
      value: "? °C"
      color: "#888888"
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "ERROR: INVALID FALLBACK MODE INDEX 0: 'bogus'",
			expectedConfig:       nil,
		},
		{
			name: "InvalidFallbackDuration",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - fallback:
      mode: "no-value-read"
      after: "1hour"
      value: "? °C"
      color: "#888888"
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "ERROR PARSING FALLBACK DURATION INDEX 0",
			expectedConfig:       nil,
		},
		{
			name: "InvalidFallbackColor",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - fallback:
      mode: "no-value-read"
      after: "1h"
      value: "? °C"
      color: "888888"
`), nil
			},
			expectedError:        true,
			expectedErrorMessage: "ERROR: INVALID FALLBACK COLOR INDEX 0: '888888'",
			expectedConfig:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock osReadFile
			osReadFile = tt.mockReadFile

			cfg, err := LoadConfig("dummy_path")
			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMessage)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, tt.expectedConfig.Mqtt.BrokerAsUri, cfg.Mqtt.BrokerAsUri)
				assert.Equal(t, tt.expectedConfig.DispatcherEntries[0].Operation, cfg.DispatcherEntries[0].Operation)
				assert.NotNil(t, cfg.DispatcherEntries[0].ColorScriptCallback)
				expectedColor, _ := tt.expectedConfig.DispatcherEntries[0].ColorScriptCallback(0)
				actualColor, _ := cfg.DispatcherEntries[0].ColorScriptCallback(0)
				assert.Equal(t, expectedColor, actualColor)
			}
		})
	}
}

func TestLoadConfigFallback(t *testing.T) {
	t.Run("ValidFallback", func(t *testing.T) {
		osReadFile = func(path string) ([]byte, error) {
			return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - fallback:
      mode: "no-value-read"
      after: "1h"
      value: "? °C"
      color: "#888888"
`), nil
		}

		cfg, err := LoadConfig("dummy_path")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		entry := cfg.DispatcherEntries[0]
		assert.True(t, entry.HasFallback())
		assert.Equal(t, time.Hour, entry.FallbackAfter)
		assert.Equal(t, "? °C", entry.Fallback.Value)
		assert.Equal(t, "#888888", entry.Fallback.Color)
	})

	t.Run("FallbackNoneSkipsValidation", func(t *testing.T) {
		osReadFile = func(path string) ([]byte, error) {
			return []byte(`
mqtt:
  broker: "tcp://localhost:1883"
dispatcher-entries:
  - fallback:
      mode: "none"
`), nil
		}

		cfg, err := LoadConfig("dummy_path")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		entry := cfg.DispatcherEntries[0]
		assert.False(t, entry.HasFallback())
		assert.Equal(t, time.Duration(0), entry.FallbackAfter)
	})
}
