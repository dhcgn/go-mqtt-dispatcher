package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	osReadFile = func(path string) ([]byte, error) {
		return os.ReadFile(path)
	}
)

func LoadConfig(path string) (*RootConfig, error) {
	data, err := osReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg RootConfig
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, err
	}

	// Parse mqtt broker as url
	cfg.Mqtt.BrokerAsUri, err = url.Parse(cfg.Mqtt.Broker)
	if err != nil {
		return nil, err
	}

	for e_i, e := range cfg.DispatcherEntries {
		if e.Operation != string(OperatorNone) && e.Operation != string(OperatorSum) {
			return nil, fmt.Errorf("ERROR: INVALID OPERATION INDEX %d: '%s'", e_i, e.Operation)
		}
	}

	for e_i, e := range cfg.DispatcherEntries {
		if e.ColorScript != "" {
			colorCallback, err := createColorCallback(e.ColorScript)
			if err != nil {
				return nil, fmt.Errorf("ERROR CREATING COLOR CALLBACK FOR CONFIG %d: %v", e_i, err)
			}
			cfg.DispatcherEntries[e_i].ColorScriptCallback = colorCallback
		}
	}

	for e_i, e := range cfg.DispatcherEntries {
		if e.Fallback == nil {
			continue
		}
		fb := e.Fallback

		switch fallbackMode(fb.Mode) {
		case FallbackModeNone, FallbackModeNoValueRead, FallbackModeNoValueChange, "":
		default:
			return nil, fmt.Errorf("ERROR: INVALID FALLBACK MODE INDEX %d: '%s'", e_i, fb.Mode)
		}

		if fb.Mode == "" || fallbackMode(fb.Mode) == FallbackModeNone {
			continue
		}

		dur, err := time.ParseDuration(fb.After)
		if err != nil {
			return nil, fmt.Errorf("ERROR PARSING FALLBACK DURATION INDEX %d: %v", e_i, err)
		}
		if dur <= 0 {
			return nil, fmt.Errorf("ERROR: FALLBACK DURATION MUST BE POSITIVE INDEX %d: '%s'", e_i, fb.After)
		}
		cfg.DispatcherEntries[e_i].FallbackAfter = dur

		if !isValidHexColor(fb.Color) {
			return nil, fmt.Errorf("ERROR: INVALID FALLBACK COLOR INDEX %d: '%s'", e_i, fb.Color)
		}
	}

	return &cfg, nil
}
