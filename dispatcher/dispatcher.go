package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/types"
	"log"
	"net/url"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/oliveagle/jsonpath"
)

type Dispatcher struct {
	config *types.Config
	state  map[string]map[string]float64
}

func NewDispatcher(config *types.Config) (*Dispatcher, error) {
	d := &Dispatcher{
		config: config,
		state:  make(map[string]map[string]float64),
	}

	// Initialize inner maps for each accumulated topic group
	for _, topicAcc := range config.TopicsAccumulated {
		d.state[topicAcc.Group] = make(map[string]float64)
	}

	return d, nil
}

type publishMessage struct {
	Text string `json:"text"`
	Icon string `json:"icon,omitempty"`
}

func (d *Dispatcher) handleMessage(topic types.TopicConfig) func(client mqtt.Client, msg mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		val, err := extractToFloat(msg.Payload(), topic.Transform)
		if err != nil {
			return
		}

		jsonData := creatingFormattedPublishMessage(val, topic.Transform.OutputFormat, topic.Icon)

		// Log
		log.Printf("TOPIC: %s Publish: %s\n", topic.Subscribe, jsonData)

		// Publish
		token := client.Publish(topic.Publish, 0, true, jsonData)
		token.Wait()
	}
}

func (d *Dispatcher) handleAccMessage(topicsAccumulated types.TopicsAccumulatedConfig, topic types.AccumulatedTopicConfig) func(client mqtt.Client, msg mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		val, err := extractToFloat(msg.Payload(), topic.Transform)
		if err != nil {
			return
		}

		d.state[topicsAccumulated.Group][topic.Subscribe] = val

		res := d.accumulation(topicsAccumulated.Operation, topicsAccumulated.Group)
		jsonData := creatingFormattedPublishMessage(res, topicsAccumulated.OutputFormat, topicsAccumulated.Icon)

		// Log
		log.Printf("TOPIC Acc: %s Publish: %s\n", topic.Subscribe, jsonData)

		// Publish
		token := client.Publish(topicsAccumulated.Publish, 0, true, jsonData)
		token.Wait()
	}
}

func (d *Dispatcher) accumulation(op, group string) float64 {
	var res float64 = 0
	if op == "sum" {
		var sum float64 = 0
		for _, v := range d.state[group] {
			sum += v
		}
		res = sum
	} else {
		log.Printf("Operation not implemented: %s", op)
		res = 0
	}
	return res
}

func creatingFormattedPublishMessage(num float64, format string, icon string) []byte {
	// Format
	formattedResult := fmt.Sprintf(format, num)

	pubMsg := publishMessage{
		Text: formattedResult,
	}

	if icon != "" {
		pubMsg.Icon = icon
	}

	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		log.Printf("Error marshaling json: %v", err)
		return []byte(`{"text": "ERR"}`)
	}

	return jsonData
}

func (d *Dispatcher) Run(ids ...string) {
	var id string
	if len(ids) > 0 {
		for _, i := range ids {
			id += i
		}
	} else {
		id = "dispatcher"
	}

	client := connect(id, d.config.Mqtt.BrokerAsUri)

	for _, topic := range d.config.Topics {
		log.Println("Subscribing to topic: ", topic.Subscribe)
		client.Subscribe(topic.Subscribe, 0, d.handleMessage(topic))
	}

	for _, topicAcc := range d.config.TopicsAccumulated {
		for _, topic := range topicAcc.Topics {
			log.Println("Subscribing to topic for accumulation: ", topic.Subscribe)
			client.Subscribe(topic.Subscribe, 0, d.handleAccMessage(topicAcc, topic))
		}
	}
}

func extractToFloat(input []byte, tranform types.Transform) (float64, error) {
	var res interface{}
	if tranform.GetJsonPath() != "" {
		var json_data interface{}
		json.Unmarshal([]byte(input), &json_data)

		var err error
		res, err = jsonpath.JsonPathLookup(json_data, tranform.GetJsonPath())
		if err != nil {
			log.Println("transformAccJsonPath JsonPath, error: ", err, " input: ", string(input), " jsonPath: ", tranform.GetJsonPath())
			return 0, err
		}
	} else {
		res = input
	}
	// Parse to float
	val, err := strconv.ParseFloat(fmt.Sprintf("%v", res), 64)
	if err != nil {
		log.Println("transformAccJsonPath ParseError, error: ", err, " input: ", string(input), " jsonPath: ", tranform.GetJsonPath())
		return 0, err
	}

	// Invert
	if tranform.GetInvert() {
		val = -val
	}

	return val, nil
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
