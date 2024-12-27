package types

import "net/url"

type MqttConfig struct {
	Broker      string `yaml:"broker"`
	BrokerAsUri *url.URL
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}

type TransformConfig struct {
	JsonPath string `yaml:"jsonPath"`
	Invert   bool   `yaml:"invert,omitempty"`

	OutputFormat string `yaml:"outputFormat"`
}

type TopicConfig struct {
	Subscribe string          `yaml:"subscribe"`
	Transform TransformConfig `yaml:"transform"`
	Publish   string          `yaml:"publish"`
	Icon      string          `yaml:"icon"`
}

type AccumulatedTopicTransform struct {
	JsonPath string `yaml:"jsonPath"`
	Invert   bool   `yaml:"invert,omitempty"`
}

type AccumulatedTopicConfig struct {
	Subscribe string                    `yaml:"subscribe"`
	Transform AccumulatedTopicTransform `yaml:"transform"`
}

type TopicsAccumulatedConfig struct {
	Group        string                   `yaml:"group"`
	Publish      string                   `yaml:"publish"`
	Icon         string                   `yaml:"icon"`
	Operation    string                   `yaml:"operation"`
	OutputFormat string                   `yaml:"outputFormat"`
	Topics       []AccumulatedTopicConfig `yaml:"topics"`
}

type Config struct {
	Mqtt              MqttConfig                `yaml:"mqtt"`
	Topics            []TopicConfig             `yaml:"topics"`
	TopicsAccumulated []TopicsAccumulatedConfig `yaml:"topics_accumulated"`
}
