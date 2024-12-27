package types

import "net/url"

type MqttConfig struct {
	Broker      string `yaml:"broker"`
	BrokerAsUri *url.URL
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}

type TransformConfig struct {
	JsonPath     string `yaml:"jsonPath"`
	Round        string `yaml:"round"`
	OutputFormat string `yaml:"outputFormat"`
}

type TopicConfig struct {
	Subscribe string          `yaml:"subscribe"`
	Transform TransformConfig `yaml:"transform"`
	Publish   string          `yaml:"publish"`
}

type Config struct {
	Mqtt   MqttConfig    `yaml:"mqtt"`
	Topics []TopicConfig `yaml:"topics"`
}
