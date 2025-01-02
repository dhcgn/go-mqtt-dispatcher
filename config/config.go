package config

import (
	"fmt"
	"net/url"
	"os"

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
			return nil, fmt.Errorf("ERROR: INVALID OPERATION %d: %s", e_i, e.Operation)
		}
	}

	// TODO Move to config validation
	for e_i, e := range cfg.DispatcherEntries {
		if e.ColorScript != "" {
			colorCallback, err := createColorCallback(e.ColorScript)
			if err != nil {
				return nil, fmt.Errorf("ERROR CREATING COLOR CALLBACK FOR CONFIG %d: %v", e_i, err)
			}
			cfg.DispatcherEntries[e_i].ColorScriptCallback = colorCallback
		}
	}

	return &cfg, nil
}
