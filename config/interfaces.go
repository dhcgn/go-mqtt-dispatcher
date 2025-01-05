package config

type TibberApiEntry interface {
	GetID() string
	GetTibberApiSource() TibberApiSource
	GetTopicsToPublish() []MqttTopicDefinition
	GetEntry() Entry
	GetName() string
	GetTypeName() string
}

type HttpEntry interface {
	GetID() string
	GetHttpSource() HttpSource
	GetTopicsToPublish() []MqttTopicDefinition
	GetSources() []HttpUrlDefinition
	GetIntervalSec() int
	GetEntry() Entry
	GetName() string
	GetTypeName() string
}

type MqttEntry interface {
	GetID() string
	GetMqttSource() MqttSource
	GetTopicsToPublish() []MqttTopicDefinition
	GetTopicsToSubscribe() []MqttTopicDefinition
	GetEntry() Entry
	GetName() string
	GetTypeName() string
}
