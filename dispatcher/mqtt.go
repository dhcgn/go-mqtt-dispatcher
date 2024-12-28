package dispatcher

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttClient interface {
	Publish(topic string, payload []byte) error
	Subscribe(topic string, callback func([]byte)) error
}

type PahoMqttClient struct {
	client mqtt.Client
}

func NewPahoMqttClient(client mqtt.Client) *PahoMqttClient {
	return &PahoMqttClient{client: client}
}

func (c *PahoMqttClient) Publish(topic string, payload []byte) error {
	log.Printf("Publishing to '%s': '%s'\n", topic, payload)
	token := c.client.Publish(topic, 0, true, payload)
	token.Wait()
	err := token.Error()
	if err != nil {
		log.Printf("Error publishing message: %v", err)
	}
	return err
}

func (c *PahoMqttClient) Subscribe(topic string, callback func([]byte)) error {
	handler := func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Payload())
	}
	token := c.client.Subscribe(topic, 0, handler)
	token.Wait()
	return token.Error()
}
