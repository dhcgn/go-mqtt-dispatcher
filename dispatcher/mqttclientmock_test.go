package dispatcher

import (
	"go.uber.org/zap"
)

type MockMqttClient struct {
	PublishedMessages map[string][]byte
	Subscriptions     map[string]func([]byte)
	logger            *zap.SugaredLogger
}

func NewMockMqttClient(logger *zap.SugaredLogger) *MockMqttClient {
	if logger == nil {
		// Create a no-op logger if none provided
		noopLogger, _ := zap.NewDevelopment()
		logger = noopLogger.Sugar()
	}

	return &MockMqttClient{
		PublishedMessages: make(map[string][]byte),
		Subscriptions:     make(map[string]func([]byte)),
		logger:            logger,
	}
}

func (m *MockMqttClient) Publish(topic string, payload []byte) error {
	m.logger.Infow("Publishing message", "topic", topic, "payload", string(payload))
	m.PublishedMessages[topic] = payload
	return nil
}

func (m *MockMqttClient) Subscribe(topic string, callback func([]byte)) error {
	m.logger.Infow("Subscribing to topic", "topic", topic)
	m.Subscriptions[topic] = callback
	return nil
}

// Helper method for testing
func (m *MockMqttClient) SimulateMessage(topic string, payload []byte) {
	m.logger.Infow("Simulating message", "topic", topic, "payload", string(payload))
	if callback, ok := m.Subscriptions[topic]; ok {
		callback(payload)
	}
}

// IsSubscribed checks if a topic has an active subscription
func (m *MockMqttClient) IsSubscribed(topic string) bool {
	_, exists := m.Subscriptions[topic]
	return exists
}
