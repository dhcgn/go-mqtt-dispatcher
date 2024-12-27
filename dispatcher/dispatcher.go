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
		// Transform
		m := msg.Payload()
		if topic.Transform.JsonPath != "" {
			m = extract(m, topic.Transform.JsonPath, topic.Transform.Invert)
		}

		val, err := strconv.ParseFloat(string(m), 64)
		if err != nil {
			log.Printf("Error parsing float: %v", err)
			return
		}

		jsonData := creatingPublishMessage(val, topic.Transform.OutputFormat, topic.Icon)

		// Log
		log.Printf("TOPIC Acc: %s Publish: %s\n", topic.Subscribe, jsonData)

		// Publish
		token := client.Publish(topic.Publish, 0, true, jsonData)
		token.Wait()
	}
}

func (d *Dispatcher) handleAccMessage(topicsAccumulated types.TopicsAccumulatedConfig, topic types.AccumulatedTopicConfig) func(client mqtt.Client, msg mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		// Transform
		m := msg.Payload()
		if topic.Transform.JsonPath != "" {
			m = extract(m, topic.Transform.JsonPath, topic.Transform.Invert)
		}

		// Set State for group item
		val, err := strconv.ParseFloat(string(m), 64)
		if err != nil {
			log.Printf("Error parsing float: %v", err)
			return
		}
		d.state[topicsAccumulated.Group][topic.Subscribe] = val

		var res float64 = 0
		// Accumulate every value from group
		if topicsAccumulated.Operation == "sum" {
			var sum float64 = 0
			for _, v := range d.state[topicsAccumulated.Group] {
				sum += v
			}
			res = sum
		} else {
			log.Printf("Operation not implemented: %s", topicsAccumulated.Operation)
			res = 0
		}

		jsonData := creatingPublishMessage(res, topicsAccumulated.OutputFormat, topicsAccumulated.Icon)

		// Log
		log.Printf("TOPIC Acc: %s Publish: %s\n", topic.Subscribe, jsonData)

		// Publish
		token := client.Publish(topicsAccumulated.Publish, 0, true, jsonData)
		token.Wait()
	}
}

func creatingPublishMessage(num float64, format string, icon string) []byte {
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
			log.Println("Subscribing to topic for accumaltion: ", topic.Subscribe)
			client.Subscribe(topic.Subscribe, 0, d.handleAccMessage(topicAcc, topic))
		}
	}
}

func extract(input []byte, jsonPath string, invert bool) []byte {
	var json_data interface{}
	json.Unmarshal([]byte(input), &json_data)

	res, err := jsonpath.JsonPathLookup(json_data, jsonPath)
	if err != nil {
		log.Println("transformAccJsonPath, error: ", err, " input: ", string(input), " jsonPath: ", jsonPath)
		return []byte("ERR")
	}

	// Invert
	if invert {
		res = -res.(float64)
	}

	return []byte(fmt.Sprintf("%v", res))
}

// func transformJsonPath(input []byte, t types.TransformConfig) []byte {
// 	var json_data interface{}
// 	json.Unmarshal([]byte(input), &json_data)

// 	res, err := jsonpath.JsonPathLookup(json_data, t.JsonPath)
// 	if err != nil {
// 		log.Println(err)
// 		return []byte("ERR")
// 	}

// 	// Invert
// 	if t.Invert {
// 		res = -res.(float64)
// 	}

// 	// Round
// 	if t.Round == "toInteger" {
// 		res = int(res.(float64))
// 	}

// 	return []byte(fmt.Sprintf(t.OutputFormat, res))
// }

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
