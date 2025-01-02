package types

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Parse mqtt broker as url
	cfg.Mqtt.BrokerAsUri, err = url.Parse(cfg.Mqtt.Broker)
	if err != nil {
		return nil, err
	}

	// TODO Move to config validation
	for e_i, e := range cfg.DispatcherEntries {
		if e.ColorScript != "" {
			colorCallback, err := createColorCallback(e.ColorScript)
			if err != nil {
				return nil, fmt.Errorf("Error creating color callback for config %d: %v", e_i, err)
			}
			cfg.DispatcherEntries[e_i].ColorScriptCallback = colorCallback
		}
	}

	return &cfg, nil
}
