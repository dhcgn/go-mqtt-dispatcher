package types

import "net/url"

type MqttConfig struct {
	Broker   string `yaml:"broker"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	// Late binding
	BrokerAsUri *url.URL
}

type TransformDefinition struct {
	JsonPath     string `yaml:"jsonPath"`
	Invert       bool   `yaml:"invert,omitempty"`
	OutputFormat string `yaml:"outputFormat,omitempty"`
}

type FilterDefinition struct {
	IgnoreLessThan *float64 `yaml:"ignore-less-than,omitempty"`
}

type MqttTopicDefinition struct {
	Topic     string              `yaml:"topic"`
	Transform TransformDefinition `yaml:"transform"`
	Filter    *FilterDefinition   `yaml:"filter,omitempty"`
}

type Entry struct {
	Name            string                `yaml:"name"`
	TopicsToPublish []MqttTopicDefinition `yaml:"topics-to-publish,omitempty"`
	Icon            string                `yaml:"icon,omitempty"`
	ColorScript     string                `yaml:"color-script,omitempty"`
	Operation       string                `yaml:"operation,omitempty"`
	Source          EntrySource           `yaml:"source,omitempty"`

	// Late binding
	ColorScriptCallback func(float64) (string, error)
}

type operator string

const (
	None operator = ""
	Sum  operator = "sum"
)

func (e Entry) MustAccumulate() (bool, operator) {
	if e.Source.HttpSource != nil {
		if len(e.Source.HttpSource.Urls) > 1 {
			return true, operator(e.Operation)
		}
	}
	if e.Source.MqttSource != nil {
		if len(e.Source.MqttSource.TopicsToSubscribe) > 1 {
			return true, operator(e.Operation)
		}
	}
	return false, None
}

type EntrySource struct {
	MqttSource *MqttSource `yaml:"mqtt,omitempty"`
	HttpSource *HttpSource `yaml:"http,omitempty"`
}

type MqttSource struct {
	TopicsToSubscribe []MqttTopicDefinition `yaml:"topics-to-subscribe,omitempty"`
}

type HttpSource struct {
	Urls        []HttpUrlDefinition `yaml:"urls"`
	IntervalSec int                 `yaml:"interval_sec"`
}

type HttpUrlDefinition struct {
	Url       string              `yaml:"url"`
	Transform TransformDefinition `yaml:"transform"`
}

type Config struct {
	Mqtt              MqttConfig `yaml:"mqtt"`
	DispatcherEntries []Entry    `yaml:"dispatcher-entries"`
}

func (t MqttTopicDefinition) GetIgnoreLessThanConfig() (hasLessThanConfig bool, lessThan float64) {
	if t.Filter == nil {
		return false, 0
	}

	if t.Filter.IgnoreLessThan == nil {
		return false, 0
	}

	return true, *t.Filter.IgnoreLessThan
}
