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
	getTicker = func(d time.Duration) *time.Ticker {
		return time.NewTicker(1 * time.Millisecond)
	}
	httpEntry := config.HttpEntryImpl{Entry: entry}
	go dispatcher.runHttp(httpEntry)

	// Wait for the ticker to tick
	time.Sleep(10 * time.Millisecond)

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
	mqttEntry := config.MqttEntryImpl{Entry: entry}
	dispatcher.runMqtt(mqttEntry)

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

func TestCallback(t *testing.T) {
	tests := []struct {
		name        string
		entry       config.Entry
		changeState func(dispatcherState)
		payload     []byte
		expected    string
		config      callbackConfig
	}{
		{
			name: "No Transform Path, No Accumulation, No Filter, No Color script, No Icon",
			entry: config.Entry{
				Name: "testEntry",
				TopicsToPublish: []config.MqttTopicDefinition{
					{Topic: "test/publish"},
				},
			},
			payload:  []byte(`42`),
			expected: `{"text":"42"}`,
			config: callbackConfig{
				Entry: config.Entry{
					Name: "testEntry",
					TopicsToPublish: []config.MqttTopicDefinition{
						{Topic: "test/publish"},
					},
				},
				Id: "test/subscribe",
				TransSource: config.MqttTopicDefinition{
					Transform: config.TransformDefinition{JsonPath: ""},
				},
				TransTarget: config.MqttTopicDefinition{
					Transform: config.TransformDefinition{},
				},
				Filter: config.MqttTopicDefinition{
					Filter: &config.FilterDefinition{IgnoreLessThan: new(float64)},
				},
			},
		},
		{
			name: "With Transform Path, Accumulation, Filter, Color script, Icon",
			entry: config.Entry{
				Name: "testEntry",
				TopicsToPublish: []config.MqttTopicDefinition{
					{Topic: "test/publish"},
				},
				Icon: "testIcon",
				ColorScriptCallback: func(v float64) (string, error) {
					return "#FFFFFF", nil
				},
				Operation: "sum",
			},
			payload:  []byte(`{"value": 42}`),
			expected: `{"text":"62","icon":"testIcon","color":"#FFFFFF"}`,
			changeState: func(state dispatcherState) {
				state["testEntry"] = map[string]float64{
					"test/subscribe_1": 10,
					"test/subscribe_2": 20,
				}
			},
			config: callbackConfig{
				Entry: config.Entry{
					Name: "testEntry",
					TopicsToPublish: []config.MqttTopicDefinition{
						{Topic: "test/publish"},
					},
					Icon: "testIcon",
					ColorScriptCallback: func(v float64) (string, error) {
						return "#FFFFFF", nil
					},
					Operation: "sum",
					Source: config.EntrySource{
						MqttSource: &config.MqttSource{
							TopicsToSubscribe: []config.MqttTopicDefinition{
								{Topic: "test/subscribe_1", Transform: config.TransformDefinition{JsonPath: "$.value"}},
								{Topic: "test/subscribe_2", Transform: config.TransformDefinition{JsonPath: "$.value"}},
							},
						},
					},
				},
				Id: "test/subscribe_1",
				TransSource: config.MqttTopicDefinition{
					Transform: config.TransformDefinition{JsonPath: "$.value"},
				},
				TransTarget: config.MqttTopicDefinition{
					Transform: config.TransformDefinition{},
				},
				Filter: config.MqttTopicDefinition{
					Filter: &config.FilterDefinition{IgnoreLessThan: new(float64)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := func(s string) {
				t.Log(s)
			}

			mqttClient := NewMockMqttClient(log)

			dispatcher, err := NewDispatcher(&[]config.Entry{tt.entry}, mqttClient, log)
			if err != nil {
				t.Fatalf("Failed to create dispatcher: %v", err)
			}

			if tt.changeState != nil {
				tt.changeState(dispatcher.state)
			}

			// Simulate receiving a message
			dispatcher.callback(tt.payload, tt.config, func(msg []byte) {
				mqttClient.Publish("test/publish", msg)
			})

			// Check if the message was published
			if msg, ok := mqttClient.PublishedMessages["test/publish"]; !ok {
				t.Errorf("Expected message to be published to test/publish")
			} else {
				if string(msg) != tt.expected {
					t.Errorf("Expected message %s, but got %s", tt.expected, string(msg))
				}
			}
		})
	}
}
