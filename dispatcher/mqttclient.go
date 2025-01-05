package dispatcher

import (
	"bytes"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type MqttClient interface {
	Publish(topic string, payload []byte) error
	Subscribe(topic string, callback func([]byte)) error
}

type PahoMqttClient struct {
	client mqtt.Client
	logger *zap.SugaredLogger
}

func NewPahoMqttClient(client mqtt.Client, logger *zap.SugaredLogger) *PahoMqttClient {
	return &PahoMqttClient{client: client, logger: logger}
}

func (c *PahoMqttClient) Publish(topic string, payload []byte) error {
	c.logger.Infow("Publish", "topic", topic, "payload", shortenPayload(payload))
	token := c.client.Publish(topic, 0, true, payload)
	token.Wait()
	err := token.Error()
	if err != nil {
		c.logger.Infof("Error publishing message: %v", err)
	}
	return err
}

func (c *PahoMqttClient) Subscribe(topic string, callback func([]byte)) error {
	c.logger.Infow("Subscribing to", "topic", topic)
	handler := func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Payload())
	}
	token := c.client.Subscribe(topic, 0, handler)
	token.Wait()
	return token.Error()
}

// shortenPayload returns a shortened version without linebreaks of the payload for logging purposes
func shortenPayload(payload []byte) string {
	// Remove line breaks
	payload = bytes.ReplaceAll(payload, []byte("\n"), []byte(" "))
	// Remove multiple whitespaces
	payload = bytes.ReplaceAll(payload, []byte("  "), []byte(" "))
	for bytes.Contains(payload, []byte("  ")) {
		payload = bytes.ReplaceAll(payload, []byte("  "), []byte(" "))
	}
	if len(payload) > 100 {
		return string(payload[:100]) + "..."
	}
	return string(payload)
}
