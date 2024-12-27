package dispatcher

import (
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
