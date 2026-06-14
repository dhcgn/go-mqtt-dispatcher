package dispatcher

import "sync"

type MockMqttClient struct {
	// mu guards all maps below; Publish/Subscribe are invoked from background
	// goroutines (HTTP/tibber pollers, fallback watchdog) while tests read state.
	mu                sync.Mutex
	PublishedMessages map[string][]byte
	PublishCount      map[string]int
	Subscriptions     map[string]func([]byte)
	Log               func(s string)
}

func NewMockMqttClient(logger ...func(s string)) *MockMqttClient {

	// If no log function is provided, use a no-op function
	if len(logger) == 0 {
		logger = append(logger, func(s string) {})
	}

	return &MockMqttClient{
		PublishedMessages: make(map[string][]byte),
		PublishCount:      make(map[string]int),
		Subscriptions:     make(map[string]func([]byte)),
		Log:               logger[0],
	}
}

func (m *MockMqttClient) Publish(topic string, payload []byte) error {
	m.Log("Publishing to '" + topic + "': '" + string(payload) + "'")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishedMessages[topic] = payload
	m.PublishCount[topic]++
	return nil
}

func (m *MockMqttClient) Subscribe(topic string, callback func([]byte)) error {
	m.Log("Subscribing to '" + topic + "'")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Subscriptions[topic] = callback
	return nil
}

// Helper method for testing
func (m *MockMqttClient) SimulateMessage(topic string, payload []byte) {
	m.Log("Simulating message for '" + topic + "'")
	// Resolve the callback under the lock, but invoke it outside to avoid
	// deadlocking with Publish (which the callback calls).
	m.mu.Lock()
	callback, ok := m.Subscriptions[topic]
	m.mu.Unlock()
	if ok {
		callback(payload)
	}
}

// IsSubscribed checks if a topic has an active subscription
func (m *MockMqttClient) IsSubscribed(topic string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.Subscriptions[topic]
	return exists
}

// GetPublishedMessage returns the last payload published to a topic.
func (m *MockMqttClient) GetPublishedMessage(topic string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.PublishedMessages[topic]
	return msg, ok
}

// GetPublishCount returns the number of times a topic was published to.
func (m *MockMqttClient) GetPublishCount(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.PublishCount[topic]
}
