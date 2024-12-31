package types

import "net/url"

type Transform interface {
	GetJsonPath() string
	GetInvert() bool
}

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

type IgnoreConfig struct {
	LessThan *float64 `yaml:"lessThan,omitempty"`
}

type TopicConfig struct {
	Subscribe string          `yaml:"subscribe"`
	Transform TransformConfig `yaml:"transform"`
	Publish   string          `yaml:"publish"`
	Icon      string          `yaml:"icon"`
	Ignore    *IgnoreConfig   `yaml:"ignore,omitempty"`
}

func (t TransformConfig) GetJsonPath() string {
	return t.JsonPath
}
func (t TransformConfig) GetInvert() bool {
	return t.Invert
}

type AccumulatedTopicTransform struct {
	JsonPath string `yaml:"jsonPath"`
	Invert   bool   `yaml:"invert,omitempty"`
}

func (t AccumulatedTopicTransform) GetJsonPath() string {
	return t.JsonPath
}

func (t AccumulatedTopicTransform) GetInvert() bool {
	return t.Invert
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
	Ignore       *IgnoreConfig            `yaml:"ignore,omitempty"`
	Topics       []AccumulatedTopicConfig `yaml:"topics"`
}

func (t TopicsAccumulatedConfig) GetIgnoreLessThanConfig() (hasLessThanConfig bool, lessThan float64) {
	if t.Ignore != nil && t.Ignore.LessThan != nil {
		return true, *t.Ignore.LessThan
	}
	return false, 0
}

func (t TopicConfig) GetIgnoreLessThanConfig() (hasLessThanConfig bool, lessThan float64) {
	if t.Ignore != nil && t.Ignore.LessThan != nil {
		return true, *t.Ignore.LessThan
	}
	return false, 0
}

type HttpConfig struct {
	Url         string          `yaml:"url"`
	IntervalSec int             `yaml:"interval_sec"`
	Transform   TransformConfig `yaml:"transform"`
	Publish     string          `yaml:"publish"`
	Icon        string          `yaml:"icon"`
}

type Config struct {
	Mqtt              MqttConfig                `yaml:"mqtt"`
	Topics            []TopicConfig             `yaml:"topics"`
	TopicsAccumulated []TopicsAccumulatedConfig `yaml:"topics_accumulated"`
	Http              []HttpConfig              `yaml:"http"`
}
