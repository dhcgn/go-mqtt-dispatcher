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
	config     *types.Config
	state      map[string]map[string]float64
	mqttClient MqttClient
}

func NewDispatcher(config *types.Config, mqttClient MqttClient) (*Dispatcher, error) {
	d := &Dispatcher{
		config:     config,
		state:      make(map[string]map[string]float64),
		mqttClient: mqttClient,
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

func (d *Dispatcher) handleMessage(topic types.TopicConfig) func([]byte) {
	return func(payload []byte) {
		val, err := extractToFloat(payload, topic.Transform)
		if err != nil {
			return
		}

		var jsonData []byte
		// Check for ignore, than delete topic with an empty payload
		if has, lt := topic.GetIgnoreLessThanConfig(); has && val < lt {
			jsonData = []byte{}
		} else {
			jsonData = creatingFormattedPublishMessage(val, topic.Transform.OutputFormat, topic.Icon)
		}

		_ = d.mqttClient.Publish(topic.Publish, jsonData)
	}
}

func (d *Dispatcher) handleAccMessage(topicsAccumulated types.TopicsAccumulatedConfig, topic types.AccumulatedTopicConfig) func([]byte) {
	return func(payload []byte) {
		val, err := extractToFloat(payload, topic.Transform)
		if err != nil {
			return
		}

		d.state[topicsAccumulated.Group][topic.Subscribe] = val
		val = d.accumulatFromStorage(topicsAccumulated.Operation, topicsAccumulated.Group)

		var jsonData []byte
		// Check for ignore, than delete topic with an empty payload
		if has, lt := topicsAccumulated.GetIgnoreLessThanConfig(); has && val < lt {
			jsonData = []byte{}
		} else {
			jsonData = creatingFormattedPublishMessage(val, topicsAccumulated.OutputFormat, topicsAccumulated.Icon)
		}

		_ = d.mqttClient.Publish(topicsAccumulated.Publish, jsonData)
	}
}

func (d *Dispatcher) accumulatFromStorage(op, group string) float64 {
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
	mqttClient := NewPahoMqttClient(client)
	d.mqttClient = mqttClient

	subscribe(d)
}

func subscribe(d *Dispatcher) {
	for _, topic := range d.config.Topics {
		log.Println("Subscribing to topic: ", topic.Subscribe)
		err := d.mqttClient.Subscribe(topic.Subscribe, d.handleMessage(topic))
		if err != nil {
			log.Printf("Error subscribing to topic %s: %v", topic.Subscribe, err)
		}
	}

	for _, topicAcc := range d.config.TopicsAccumulated {
		for _, topic := range topicAcc.Topics {
			log.Println("Subscribing to topic for accumulation: ", topic.Subscribe)
			err := d.mqttClient.Subscribe(topic.Subscribe, d.handleAccMessage(topicAcc, topic))
			if err != nil {
				log.Printf("Error subscribing to topic %s: %v", topic.Subscribe, err)
			}
		}
	}
}

func extractToFloat(input []byte, tranform types.Transform) (float64, error) {
	var res interface{}
	var err error
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
		res, err = strconv.ParseFloat(string(input), 64)
		if err != nil {
			log.Println("transformAccJsonPath ParseError, error: ", err, " input: ", string(input), " jsonPath: ", tranform.GetJsonPath())
			return 0, err
		}
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
