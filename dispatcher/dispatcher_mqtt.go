package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/types"
	"net/url"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/oliveagle/jsonpath"
)

type Dispatcher struct {
	entries    *[]types.MqttEntry
	state      map[string]map[string]float64
	mqttClient MqttClient
	log        func(string)
}

func NewDispatcherMqtt(mqttEntries *[]types.MqttEntry, mqttClient MqttClient, log func(string)) (*Dispatcher, error) {
	if log == nil {
		log = func(s string) {}
	}

	d := &Dispatcher{
		entries:    mqttEntries,
		state:      make(map[string]map[string]float64),
		mqttClient: mqttClient,
		log:        log,
	}

	// Initialize inner maps for each accumulated topic group
	for _, entry := range *mqttEntries {
		d.state[entry.Name] = make(map[string]float64)
	}

	// Check ColorScript in each types.HttpConfig an set callback
	for entry_id, entry := range *mqttEntries {
		if entry.ColorScript != "" {
			colorCallback, err := createColorCallback(entry.ColorScript)
			if err != nil {
				log(fmt.Sprintf("Error creating color callback for config %d: %v", entry_id, err))
			}
			(*mqttEntries)[entry_id].ColorScriptCallback = colorCallback
		}
	}

	return d, nil
}

func (d *Dispatcher) handleMessage(topic types.MqttTopicDefinition, accu bool) func([]byte) {
	return func(payload []byte) {
		val, err := extractToFloat(payload, topic.Transform, d.log)
		if err != nil {
			return
		}

		var jsonData []byte
		// Check for ignore, than delete topic with an empty payload
		if has, lt := topic.GetIgnoreLessThanConfig(); has && val < lt {
			jsonData = []byte{}
		} else {
			jsonData = creatingFormattedPublishMessage(val, topic.Transform.OutputFormat, topic.Icon, topic.ColorScriptCallback, d.log)
		}

		_ = d.mqttClient.Publish(topic.Publish, jsonData)
	}
}

func (d *Dispatcher) handleAccMessage(topicsAccumulated types.TopicsAccumulatedConfig, topic types.AccumulatedTopicConfig) func([]byte) {
	return func(payload []byte) {
		val, err := extractToFloat(payload, topic.Transform, d.log)
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
			jsonData = creatingFormattedPublishMessage(val, topicsAccumulated.OutputFormat, topicsAccumulated.Icon, topicsAccumulated.ColorScriptCallback, d.log)
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
		d.log(fmt.Sprintf("Operation not implemented: %s", op))
		res = 0
	}
	return res
}

func creatingFormattedPublishMessage(num float64, format string, icon string, color func(value float64) (string, error), log func(string)) []byte {
	// Format
	formattedResult := fmt.Sprintf(format, num)

	pubMsg := publishMessage{
		Text: formattedResult,
	}

	if icon != "" {
		pubMsg.Icon = icon
	}

	if color != nil {
		c, err := color(num)
		if err != nil {
			log(fmt.Sprintf("Error running color script: %v", err))
		} else {
			pubMsg.Color = c
		}
	}

	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		log(fmt.Sprintf("Error marshaling json: %v", err))
		return []byte(`{"text": "ERR"}`)
	}

	return jsonData
}

func (d *Dispatcher) Run(ids ...string) {
	subscribe(d)
}

func subscribe(d *Dispatcher) {
	for _, entry := range *d.entries {
		name := entry.Name
		accumulated := len(entry.TopicsToSubscribe) > 1
		for _, t := range entry.TopicsToSubscribe {
			d.log(fmt.Sprintf("%s: Subscribing to topic for accumulation: %s", name, t.Topic))

			for _, t := range entry.TopicsToSubscribe {
				err := d.mqttClient.Subscribe(t.Topic, d.handleMessage(t, accumulated))
				if err != nil {
					d.log(fmt.Sprintf("Error subscribing to topic %s: %v", t.Topic, err))
				}
			}
		}
	}
}

func extractToFloat(input []byte, tranform types.Transform, log func(string)) (float64, error) {
	var res interface{}
	var err error
	if tranform.GetJsonPath() != "" {
		var json_data interface{}
		json.Unmarshal([]byte(input), &json_data)

		var err error
		res, err = jsonpath.JsonPathLookup(json_data, tranform.GetJsonPath())
		if err != nil {
			log(fmt.Sprintf("transformAccJsonPath JsonPath error: %v input: %s jsonPath: %s", err, string(input), tranform.GetJsonPath()))
			return 0, err
		}
	} else {
		res, err = strconv.ParseFloat(string(input), 64)
		if err != nil {
			log(fmt.Sprintf("transformAccJsonPath ParseError: %v input: %s jsonPath: %s", err, string(input), tranform.GetJsonPath()))
			return 0, err
		}
	}
	// Parse to float
	val, err := strconv.ParseFloat(fmt.Sprintf("%v", res), 64)
	if err != nil {
		log(fmt.Sprintf("transformAccJsonPath ParseError: %v input: %s jsonPath: %s", err, string(input), tranform.GetJsonPath()))
		return 0, err
	}

	// Invert
	if tranform.GetInvert() {
		val = -val
	}

	return val, nil
}

func connect(clientId string, uri *url.URL, log func(string)) mqtt.Client {
	opts := createClientOptions(clientId, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log(fmt.Sprintf("mqtt connection error: %v", err))
		panic(err)
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
