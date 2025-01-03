package dispatcher

type MockMqttClient struct {
	PublishedMessages map[string][]byte
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
		Subscriptions:     make(map[string]func([]byte)),
		Log:               logger[0],
	}
}

func (m *MockMqttClient) Publish(topic string, payload []byte) error {
	m.Log("Publishing to '" + topic + "': '" + string(payload) + "'")
	m.PublishedMessages[topic] = payload
	return nil
}

func (m *MockMqttClient) Subscribe(topic string, callback func([]byte)) error {
	m.Log("Subscribing to '" + topic + "'")
	m.Subscriptions[topic] = callback
	return nil
}

// Helper method for testing
func (m *MockMqttClient) SimulateMessage(topic string, payload []byte) {
	m.Log("Simulating message for '" + topic + "'")
	if callback, ok := m.Subscriptions[topic]; ok {
		callback(payload)
	}
}

// IsSubscribed checks if a topic has an active subscription
func (m *MockMqttClient) IsSubscribed(topic string) bool {
	_, exists := m.Subscriptions[topic]
	return exists
}
