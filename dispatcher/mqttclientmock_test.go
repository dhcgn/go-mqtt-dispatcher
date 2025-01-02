package dispatcher

type MockMqttClient struct {
	PublishedMessages map[string][]byte
	Subscriptions     map[string]func([]byte)
}

func NewMockMqttClient() *MockMqttClient {
	return &MockMqttClient{
		PublishedMessages: make(map[string][]byte),
		Subscriptions:     make(map[string]func([]byte)),
	}
}

func (m *MockMqttClient) Publish(topic string, payload []byte) error {
	m.PublishedMessages[topic] = payload
	return nil
}

func (m *MockMqttClient) Subscribe(topic string, callback func([]byte)) error {
	m.Subscriptions[topic] = callback
	return nil
}

// Helper method for testing
func (m *MockMqttClient) SimulateMessage(topic string, payload []byte) {
	if callback, ok := m.Subscriptions[topic]; ok {
		callback(payload)
	}
}

// IsSubscribed checks if a topic has an active subscription
func (m *MockMqttClient) IsSubscribed(topic string) bool {
	_, exists := m.Subscriptions[topic]
	return exists
}
