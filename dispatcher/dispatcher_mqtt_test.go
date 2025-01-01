package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/types"
	"testing"
)

func TestAccumulatFromStorage(t *testing.T) {
	mockClient := NewMockMqttClient()
	config := &types.Config{}
	d, err := NewDispatcherMqtt(config, mockClient, func(s string) {})
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	// Test case 1: Sum operation with multiple values
	d.state["group1"] = map[string]float64{
		"topic1": 1.0,
		"topic2": 2.0,
		"topic3": 3.0,
	}
	result := d.accumulatFromStorage("sum", "group1")
	expected := 6.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// Test case 2: Sum operation with empty group
	d.state["group2"] = map[string]float64{}
	result = d.accumulatFromStorage("sum", "group2")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for empty group, got %v", result)
	}

	// Test case 3: Unsupported operation
	result = d.accumulatFromStorage("unsupported", "group1")
	if result != 0.0 {
		t.Errorf("Expected 0.0 for unsupported operation, got %v", result)
	}
}

func TestCreatingFormattedPublishMessage(t *testing.T) {
	tests := []struct {
		num    float64
		format string
		icon   string
		want   string
	}{
		{num: 123.456, format: "%.2f", icon: "", want: `{"text":"123.46"}`},
		{num: 123.456, format: "%.2f", icon: "icon1", want: `{"text":"123.46","icon":"icon1"}`},
		{num: 0, format: "%.0f", icon: "icon2", want: `{"text":"0","icon":"icon2"}`},
		{num: -123.456, format: "%.1f", icon: "", want: `{"text":"-123.5"}`},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("num=%v,format=%s,icon=%s", tt.num, tt.format, tt.icon), func(t *testing.T) {
			got := creatingFormattedPublishMessage(tt.num, tt.format, tt.icon, nil, func(s string) {})
			if string(got) != tt.want {
				t.Errorf("creatingFormattedPublishMessage() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestNewDispatcher(t *testing.T) {
	mockClient := NewMockMqttClient()

	// Test case 1: Valid configuration with accumulated topics
	config := &types.Config{
		TopicsAccumulated: []types.TopicsAccumulatedConfig{
			{Group: "group1"},
			{Group: "group2"},
		},
	}

	d, err := NewDispatcherMqtt(config, mockClient, func(s string) {})
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	if d.entries != config {
		t.Errorf("Expected config %v, got %v", config, d.entries)
	}

	if len(d.state) != 2 {
		t.Errorf("Expected state map length 2, got %d", len(d.state))
	}

	for _, group := range []string{"group1", "group2"} {
		if _, exists := d.state[group]; !exists {
			t.Errorf("Expected group %s to be initialized in state map", group)
		}
	}

	// Test case 2: Empty configuration
	emptyConfig := &types.Config{}
	d, err = NewDispatcherMqtt(emptyConfig, mockClient, func(s string) {})
	if err != nil {
		t.Fatalf("Failed to create dispatcher with empty config: %v", err)
	}

	if len(d.state) != 0 {
		t.Errorf("Expected empty state map, got length %d", len(d.state))
	}
}

func TestHandleMessage(t *testing.T) {
	var lessThan float64 = 2.0

	tests := []struct {
		name     string
		payload  []byte
		topic    types.TopicConfig
		wantText string
		wantIcon string
	}{
		{
			name:    "basic number",
			payload: []byte("42.5"),
			topic: types.TopicConfig{
				Subscribe: "test/topic",
				Publish:   "test/output",
				Transform: types.TransformConfig{
					OutputFormat: "%.2f",
				},
				Icon: "test-icon",
			},
			wantText: "42.50",
			wantIcon: "test-icon",
		},
		{
			name:    "ignore less than threshold",
			payload: []byte("1.5"),
			topic: types.TopicConfig{
				Subscribe: "test/topic",
				Publish:   "test/output",
				Transform: types.TransformConfig{
					OutputFormat: "%.2f",
				},
				Ignore: &types.IgnoreConfig{
					LessThan: &lessThan,
				},
			},
			wantText: "",
			wantIcon: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockMqttClient()
			config := &types.Config{
				Topics: []types.TopicConfig{tt.topic},
			}
			d, _ := NewDispatcherMqtt(config, mockClient, func(s string) {})

			// Subscribe to topics
			err := d.mqttClient.Subscribe(tt.topic.Subscribe, d.handleMessage(tt.topic))
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}

			// Simulate message
			mockClient.SimulateMessage(tt.topic.Subscribe, tt.payload)

			// Check published message
			published := mockClient.PublishedMessages[tt.topic.Publish]
			if len(published) == 0 && tt.wantText != "" {
				t.Errorf("Expected published message, got none")
				return
			}

			if len(published) > 0 {
				var msg publishMessage
				if err := json.Unmarshal(published, &msg); err != nil {
					t.Errorf("Failed to unmarshal published message: %v", err)
					return
				}

				if msg.Text != tt.wantText {
					t.Errorf("Expected text '%s', got '%s'", tt.wantText, msg.Text)
				}

				if msg.Icon != tt.wantIcon {
					t.Errorf("Expected icon '%s', got '%s'", tt.wantIcon, msg.Icon)
				}
			}
		})
	}
}

func TestHandleAccMessage(t *testing.T) {
	mockClient := NewMockMqttClient()
	config := &types.Config{
		TopicsAccumulated: []types.TopicsAccumulatedConfig{
			{
				Group:        "test_group",
				Operation:    "sum",
				Publish:      "test/acc/output",
				OutputFormat: "%.1f",
				Topics: []types.AccumulatedTopicConfig{
					{Subscribe: "test/acc/topic1"},
					{Subscribe: "test/acc/topic2"},
				},
			},
		},
	}

	d, _ := NewDispatcherMqtt(config, mockClient, func(s string) {})

	// Subscribe to topics
	for _, topicAcc := range config.TopicsAccumulated {
		for _, topic := range topicAcc.Topics {
			err := d.mqttClient.Subscribe(topic.Subscribe, d.handleAccMessage(topicAcc, topic))
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}
		}
	}

	// Send messages to accumulated topics
	mockClient.SimulateMessage("test/acc/topic1", []byte("10.5"))
	mockClient.SimulateMessage("test/acc/topic2", []byte("20.5"))

	// Check final accumulated result
	published := mockClient.PublishedMessages["test/acc/output"]
	var msg publishMessage
	if err := json.Unmarshal(published, &msg); err != nil {
		t.Errorf("Failed to unmarshal published message: %v", err)
		return
	}

	if msg.Text != "31.0" {
		t.Errorf("Expected accumulated sum '31.0', got '%s'", msg.Text)
	}
}

func TestSubscribe(t *testing.T) {
	mockClient := NewMockMqttClient()
	config := &types.Config{
		Topics: []types.TopicConfig{
			{Subscribe: "test/topic1"},
			{Subscribe: "test/topic2"},
		},
		TopicsAccumulated: []types.TopicsAccumulatedConfig{
			{
				Group:     "group1",
				Operation: "sum",
				Topics: []types.AccumulatedTopicConfig{
					{Subscribe: "test/acc/topic1"},
					{Subscribe: "test/acc/topic2"},
				},
			},
		},
	}

	d, err := NewDispatcherMqtt(config, mockClient, func(s string) {})
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	subscribe(d)

	// Check if all topics are subscribed
	for _, topic := range config.Topics {
		if !mockClient.IsSubscribed(topic.Subscribe) {
			t.Errorf("Expected subscription to topic %s", topic.Subscribe)
		}
	}

	for _, topicAcc := range config.TopicsAccumulated {
		for _, topic := range topicAcc.Topics {
			if !mockClient.IsSubscribed(topic.Subscribe) {
				t.Errorf("Expected subscription to accumulated topic %s", topic.Subscribe)
			}
		}
	}
}

func TestExtractToFloat(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		transform types.Transform
		want      float64
		wantErr   bool
	}{
		{
			name:      "basic number",
			input:     []byte("42.5"),
			transform: types.TransformConfig{},
			want:      42.5,
			wantErr:   false,
		},
		{
			name:  "json path extraction",
			input: []byte(`{"value": 42.5}`),
			transform: types.TransformConfig{
				JsonPath: "$.value",
			},
			want:    42.5,
			wantErr: false,
		},
		{
			name:  "json path extraction with invert",
			input: []byte(`{"value": 42.5}`),
			transform: types.TransformConfig{
				JsonPath: "$.value",
				Invert:   true,
			},
			want:    -42.5,
			wantErr: false,
		},
		{
			name:      "invalid number",
			input:     []byte("invalid"),
			transform: types.TransformConfig{},
			want:      0,
			wantErr:   true,
		},
		{
			name:  "invalid json path",
			input: []byte(`{"value": 42.5}`),
			transform: types.TransformConfig{
				JsonPath: "$.invalid",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractToFloat(tt.input, tt.transform, func(s string) {})
			if (err != nil) != tt.wantErr {
				t.Errorf("extractToFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractToFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
