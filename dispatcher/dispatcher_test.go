package dispatcher

import (
	"errors"
	"go-mqtt-dispatcher/config"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRunHttp(t *testing.T) {
	// Mock HTTP response
	httpGet = func(url string) (resp *http.Response, err error) {
		if url == "http://example.com" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"value": 42}`)),
			}, nil
		}
		return nil, errors.New("not found")
	}

	// Mock configuration
	entry := config.Entry{
		Name: "testEntry",
		Source: config.EntrySource{
			HttpSource: &config.HttpSource{
				Urls: []config.HttpUrlDefinition{
					{Url: "http://example.com", Transform: config.TransformDefinition{JsonPath: "$.value"}},
				},
				IntervalSec: 1,
			},
		},
		TopicsToPublish: []config.MqttTopicDefinition{
			{Topic: "test/topic"},
		},
	}

	mqttClient := NewMockMqttClient()
	log := func(s string) {
		t.Log(s)
	}

	dispatcher, err := NewDispatcher(&[]config.Entry{entry}, mqttClient, log)
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	// Run the dispatcher
	interruptRunHttpTickerAfterTick = true
	go dispatcher.runHttp(entry)

	// Wait for the ticker to tick
	time.Sleep(1 * time.Second)
	time.Sleep(100 * time.Millisecond)

	// Check if the message was published
	if msg, ok := mqttClient.PublishedMessages["test/topic"]; !ok {
		t.Errorf("Expected message to be published to test/topic")
	} else {
		expectedMsg := `{"text":"42"}`
		if string(msg) != expectedMsg {
			t.Errorf("Expected message %s, but got %s", expectedMsg, string(msg))
		}
	}
}

func TestRunMqtt(t *testing.T) {
	// Mock configuration
	entry := config.Entry{
		Name: "testEntry",
		Source: config.EntrySource{
			MqttSource: &config.MqttSource{
				TopicsToSubscribe: []config.MqttTopicDefinition{
					{Topic: "test/subscribe", Transform: config.TransformDefinition{JsonPath: "$.value"}},
				},
			},
		},
		TopicsToPublish: []config.MqttTopicDefinition{
			{Topic: "test/publish"},
		},
	}

	log := func(s string) {
		t.Log(s)
	}

	mqttClient := NewMockMqttClient(log)

	dispatcher, err := NewDispatcher(&[]config.Entry{entry}, mqttClient, log)
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	// Run the dispatcher
	dispatcher.runMqtt(entry)

	// Check if the subscription was made
	if !mqttClient.IsSubscribed("test/subscribe") {
		t.Errorf("Expected subscription to test/subscribe")
	}

	// Simulate receiving a message
	payload := []byte(`{"value": 42}`)
	mqttClient.SimulateMessage("test/subscribe", payload)

	// Check if the message was published
	if msg, ok := mqttClient.PublishedMessages["test/publish"]; !ok {
		t.Errorf("Expected message to be published to test/publish")
	} else {
		if string(msg) != `{"text":"42"}` {
			t.Errorf("Expected message %s, but got %s", string(payload), string(msg))
		}
	}
}
