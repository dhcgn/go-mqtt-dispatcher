// TODO: Evalaute all interfaces
// TODO: Update docs for new config!
package config

import "net/url"

type RootConfig struct {
	Mqtt              MqttConfig `yaml:"mqtt"`
	DispatcherEntries []Entry    `yaml:"dispatcher-entries"`
}

type MqttConfig struct {
	Broker   string `yaml:"broker"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	// Late binding
	BrokerAsUri *url.URL
}

type Entry struct {
	Name            string                `yaml:"name"`
	Disabled        bool                  `yaml:"disabled,omitempty"`
	TopicsToPublish []MqttTopicDefinition `yaml:"topics-to-publish,omitempty"`
	Icon            string                `yaml:"icon,omitempty"`
	ColorScript     string                `yaml:"color-script,omitempty"`
	Operation       string                `yaml:"operation,omitempty"`
	Source          EntrySource           `yaml:"source,omitempty"`

	// Late binding
	ColorScriptCallback func(float64) (string, error)
}

type MqttTopicDefinition struct {
	Topic     string              `yaml:"topic"`
	Transform TransformDefinition `yaml:"transform"`
	Filter    *FilterDefinition   `yaml:"filter,omitempty"`
}

type TransformDefinition struct {
	JsonPath            string `yaml:"jsonPath"`
	Invert              bool   `yaml:"invert,omitempty"`
	OutputFormat        string `yaml:"outputFormat,omitempty"`
	OutputAsTibberGraph bool   `yaml:"output-as-tibber-graph,omitempty"`
}

type FilterDefinition struct {
	IgnoreLessThan *float64 `yaml:"ignore-less-than,omitempty"`
}

type EntrySource struct {
	MqttSource      *MqttSource      `yaml:"mqtt,omitempty"`
	HttpSource      *HttpSource      `yaml:"http,omitempty"`
	TibberApiSource *TibberApiSource `yaml:"tibber-api,omitempty"`
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

type TibberApiSource struct {
	TibberApiKey string              `yaml:"tibber-api-key"`
	GraphqlQuery string              `yaml:"graphql-query"`
	IntervalSec  int                 `yaml:"interval_sec"`
	Transform    TransformDefinition `yaml:"transform"`
}

type TransformTarget interface {
	GetOutputFormat() string
	GetOutputAsTibberGraph() bool
}

func (m MqttTopicDefinition) GetOutputFormat() string {
	return m.Transform.OutputFormat
}

func (h HttpUrlDefinition) GetOutputFormat() string {
	return h.Transform.OutputFormat
}

func (m MqttTopicDefinition) GetOutputAsTibberGraph() bool {
	return m.Transform.OutputAsTibberGraph
}

func (h HttpUrlDefinition) GetOutputAsTibberGraph() bool {
	return h.Transform.OutputAsTibberGraph
}

type Filter interface {
	GetFilter() *FilterDefinition
}

func (m MqttTopicDefinition) GetFilter() *FilterDefinition {
	return m.Filter
}

type TransformSource interface {
	GetJsonPath() string
	GetInvert() bool
}

type Transformers interface {
	TransformSource
	TransformTarget
}

func (m MqttTopicDefinition) GetJsonPath() string {
	return m.Transform.JsonPath
}

func (m MqttTopicDefinition) GetInvert() bool {
	return m.Transform.Invert
}

func (h HttpUrlDefinition) GetJsonPath() string {
	return h.Transform.JsonPath
}

func (h HttpUrlDefinition) GetInvert() bool {
	return h.Transform.Invert
}

func (t TibberApiSource) GetJsonPath() string {
	return t.Transform.JsonPath
}

func (t TibberApiSource) GetInvert() bool {
	return t.Transform.Invert
}

type operator string

const (
	OperatorNone operator = ""
	OperatorSum  operator = "sum"
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
	return false, OperatorNone
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

func (entry Entry) GetID() string {
	if entry.Source.MqttSource != nil {
		mqttEntry := MqttEntryImpl{Entry: entry}
		return mqttEntry.GetID()
	} else if entry.Source.HttpSource != nil {
		httpEntry := HttpEntryImpl{Entry: entry}
		return httpEntry.GetID()
	} else if entry.Source.TibberApiSource != nil {
		tibberApiEntry := TibberApiEntryImpl{Entry: entry}
		return tibberApiEntry.GetID()
	}
	return ""
}
