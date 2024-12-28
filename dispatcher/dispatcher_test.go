package dispatcher

import (
	"fmt"
	"go-mqtt-dispatcher/types"
	"testing"
)

func TestAccumulatFromStorage(t *testing.T) {
	config := &types.Config{}
	d, err := NewDispatcher(config)
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

	// Test case 2: Sum operation with no values
	d.state["group2"] = map[string]float64{}
	result = d.accumulatFromStorage("sum", "group2")
	expected = 0.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// Test case 3: Unsupported operation
	result = d.accumulatFromStorage("unsupported", "group1")
	expected = 0.0
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
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
			got := creatingFormattedPublishMessage(tt.num, tt.format, tt.icon)
			if string(got) != tt.want {
				t.Errorf("creatingFormattedPublishMessage() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestNewDispatcher(t *testing.T) {
	// Test case 1: Valid configuration with accumulated topics
	config := &types.Config{
		TopicsAccumulated: []types.TopicsAccumulatedConfig{
			{Group: "group1"},
			{Group: "group2"},
		},
	}
	d, err := NewDispatcher(config)
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	if d.config != config {
		t.Errorf("Expected config %v, got %v", config, d.config)
	}

	if len(d.state) != 2 {
		t.Errorf("Expected state map length 2, got %d", len(d.state))
	}

	if _, exists := d.state["group1"]; !exists {
		t.Errorf("Expected group1 to be initialized in state map")
	}

	if _, exists := d.state["group2"]; !exists {
		t.Errorf("Expected group2 to be initialized in state map")
	}

	// Test case 2: Valid configuration with no accumulated topics
	config = &types.Config{}
	d, err = NewDispatcher(config)
	if err != nil {
		t.Fatalf("Failed to create dispatcher: %v", err)
	}

	if d.config != config {
		t.Errorf("Expected config %v, got %v", config, d.config)
	}

	if len(d.state) != 0 {
		t.Errorf("Expected state map length 0, got %d", len(d.state))
	}
}
