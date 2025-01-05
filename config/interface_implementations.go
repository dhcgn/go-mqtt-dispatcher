package config

import (
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

// Interface guards
var _ TibberApiEntry = (*TibberApiEntryImpl)(nil)
var _ HttpEntry = (*HttpEntryImpl)(nil)
var _ MqttEntry = (*MqttEntryImpl)(nil)

// TibberApiEntryImpl

type TibberApiEntryImpl struct {
	Entry
}

func (e TibberApiEntryImpl) GetID() string {
	return HashStrings(8, e.Name, e.Source.TibberApiSource.Transform.JsonPath)
}

func (e TibberApiEntryImpl) GetTibberApiSource() TibberApiSource {
	return *e.Source.TibberApiSource
}

func (e TibberApiEntryImpl) GetTopicsToPublish() []MqttTopicDefinition {
	return e.TopicsToPublish
}

func (e TibberApiEntryImpl) GetEntry() Entry {
	return e.Entry
}

func (e TibberApiEntryImpl) GetName() string {
	return e.Name
}

func (e TibberApiEntryImpl) GetTypeName() string {
	return "TibberApi"
}

// HttpEntryImpl

type HttpEntryImpl struct {
	Entry
}

func (e HttpEntryImpl) GetID() string {
	return HashStrings(8, e.Name, e.Source.HttpSource.Urls[0].Url)
}

func (e HttpEntryImpl) GetHttpSource() HttpSource {
	return *e.Source.HttpSource
}

func (e HttpEntryImpl) GetTopicsToPublish() []MqttTopicDefinition {
	return e.TopicsToPublish
}

func (e HttpEntryImpl) GetEntry() Entry {
	return e.Entry
}

func (e HttpEntryImpl) GetName() string {
	return e.Name
}

func (e HttpEntryImpl) GetSources() []HttpUrlDefinition {
	return e.Source.HttpSource.Urls
}

func (e HttpEntryImpl) GetIntervalSec() int {
	return e.Source.HttpSource.IntervalSec
}

func (e HttpEntryImpl) GetTypeName() string {
	return "Http"
}

// MqttEntryImpl

type MqttEntryImpl struct {
	Entry
}

func (e MqttEntryImpl) GetID() string {
	return HashStrings(8, e.Name, e.Source.MqttSource.TopicsToSubscribe[0].Topic)
}

func (e MqttEntryImpl) GetMqttSource() MqttSource {
	return *e.Source.MqttSource
}

func (e MqttEntryImpl) GetTopicsToPublish() []MqttTopicDefinition {
	return e.TopicsToPublish
}

func (e MqttEntryImpl) GetEntry() Entry {
	return e.Entry
}

func (e MqttEntryImpl) GetName() string {
	return e.Name
}

func (e MqttEntryImpl) GetTopicsToSubscribe() []MqttTopicDefinition {
	return e.Source.MqttSource.TopicsToSubscribe
}

func (e MqttEntryImpl) GetTypeName() string {
	return "Mqtt"
}

// HashStrings hashes concat strings with sha256 and returns the first length characters of base32
func HashStrings(length int, s ...string) string {
	concatenated := strings.Join(s, "")
	hash := sha256.Sum256([]byte(concatenated))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return encoded[:length]
}
