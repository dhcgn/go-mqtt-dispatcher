package types

import "net/url"

type MqttConfig struct {
	Broker      string `yaml:"broker"`
	BrokerAsUri *url.URL
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}

type TransformDefinition struct {
	JsonPath     string `yaml:"jsonPath"`
	Invert       bool   `yaml:"invert,omitempty"`
	OutputFormat string `yaml:"outputFormat,omitempty"`
}

type MqttTopicDefinition struct {
	Topic     string              `yaml:"topic"`
	Transform TransformDefinition `yaml:"transform"`
}

type MqttEntry struct {
	// Base
	Name                string                `yaml:"name"`
	TopicsToPublish     []MqttTopicDefinition `yaml:"topics-to-publish,omitempty"`
	Icon                string                `yaml:"icon,omitempty"`
	ColorScript         string                `yaml:"color-script,omitempty"`
	ColorScriptCallback func(float64) (string, error)
	Operation           string `yaml:"operation,omitempty"`

	// Mqtt
	TopicsToSubscribe []MqttTopicDefinition `yaml:"topics-to-subscribe,omitempty"`
}

type HttpUrlDefinition struct {
	Url       string              `yaml:"url"`
	Transform TransformDefinition `yaml:"transform"`
}

type HttpEntry struct {
	// Base
	Name                string                `yaml:"name"`
	TopicsToPublish     []MqttTopicDefinition `yaml:"topics-to-publish,omitempty"`
	Icon                string                `yaml:"icon,omitempty"`
	ColorScript         string                `yaml:"color-script,omitempty"`
	ColorScriptCallback func(float64) (string, error)
	Operation           string `yaml:"operation,omitempty"`

	// Http
	Urls        []HttpUrlDefinition `yaml:"urls"`
	IntervalSec int                 `yaml:"interval_sec"`
}

type DispatcherConfig struct {
	Mqtt []MqttEntry `yaml:"mqtt"`
	Http []HttpEntry `yaml:"http"`
}

type Config struct {
	Mqtt             MqttConfig       `yaml:"mqtt"`
	DispatcherConfig DispatcherConfig `yaml:"dispatcher-config"`
}

func (t MqttEntry) GetIgnoreLessThanConfig() (hasLessThanConfig bool, lessThan float64) {
	if t.Ignore != nil && t.Ignore.LessThan != nil {
		return true, *t.Ignore.LessThan
	}
	return false, 0
}
