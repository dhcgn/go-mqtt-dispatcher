package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/types"
	"log"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/oliveagle/jsonpath"
)

type Dispatcher struct {
	config *types.Config
}

func NewDispatcher(config *types.Config) (*Dispatcher, error) {
	d := &Dispatcher{
		config: config,
	}

	return d, nil
}

type publishMessage struct {
	Text string `json:"text"`
}

func (d *Dispatcher) Run() {

	client := connect("dispatcher", d.config.Mqtt.BrokerAsUri)

	for _, topic := range d.config.Topics {
		t := topic
		client.Subscribe(t.Subscribe, 0, func(client mqtt.Client, msg mqtt.Message) {
			// Transform
			m := msg.Payload()
			if t.Transform.JsonPath != "" {
				m = transformJsonPath(m, t.Transform)
			}

			// Log
			log.Printf("TOPIC: %s MSG: %s\n", t.Subscribe, m)

			// Create data structure for publish
			pubMsg := publishMessage{
				Text: string(m),
			}
			jsonData, err := json.Marshal(pubMsg)
			if err != nil {
				log.Printf("Error marshaling json: %v", err)
				return
			}

			// Publish
			token := client.Publish(t.Publish, 0, true, jsonData)
			token.Wait()
		})
	}

}

func transformJsonPath(input []byte, t types.TransformConfig) []byte {
	var json_data interface{}
	json.Unmarshal([]byte(input), &json_data)

	res, err := jsonpath.JsonPathLookup(json_data, t.JsonPath)
	if err != nil {
		log.Println(err)
		return []byte("ERR")
	}

	// Round
	if t.Round == "toInteger" {
		res = int(res.(float64))
	}

	return []byte(fmt.Sprintf(t.OutputFormat, res))
}

func connect(clientId string, uri *url.URL) mqtt.Client {
	opts := createClientOptions(clientId, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func createClientOptions(clientId string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	// opts.SetUsername(uri.User.Username())
	// password, _ := uri.User.Password()
	// opts.SetPassword(password)
	opts.SetClientID(clientId)
	return opts
}
