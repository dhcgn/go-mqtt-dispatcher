// TODO: Update comments of funcs
package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/types"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/oliveagle/jsonpath"
)

type Dispatcher struct {
	entries    *[]types.Entry
	state      map[string]map[string]float64
	mqttClient MqttClient
	log        func(string)
}

func NewDispatcher(entries *[]types.Entry, mqttClient MqttClient, log func(s string)) (*Dispatcher, error) {
	if log == nil {
		log = func(s string) {}
	}

	// TODO Move to config validation
	for e_i, e := range *entries {
		if e.ColorScript != "" {
			colorCallback, err := createColorCallback(e.ColorScript)
			if err != nil {
				log(fmt.Sprintf("Error creating color callback for config %d: %v", e_i, err))
			}
			(*entries)[e_i].ColorScriptCallback = colorCallback
		}
	}

	return &Dispatcher{
		entries:    entries,
		state:      make(map[string]map[string]float64),
		mqttClient: mqttClient,
		log:        log,
	}, nil
}

// Run starts the dispatcher
//
// 1. Iterate over all entries
// 2. Create a event listener for each type of MqttSource or HttpSource
// 3. Create a callback function for each event listener
func (d *Dispatcher) Run() {
	for _, entry := range *d.entries {
		if entry.Source.MqttSource != nil {
			d.log("Entry for mqtt: " + entry.Name)
			for _, topicSub := range entry.Source.MqttSource.TopicsToSubscribe {
				d.log("- Subscribing to " + topicSub.Topic)
				err := d.mqttClient.Subscribe(topicSub.Topic, func(payload []byte) {
					d.log("Received payload for " + topicSub.Topic)
					for _, topicPub := range entry.TopicsToPublish {
						d.callback(payload, entry, topicSub.Topic, topicSub, topicPub, topicPub, func(msg []byte) {
							d.mqttClient.Publish(topicPub.Topic, msg)
						})
					}
				})
				if err != nil {
					d.log("Error subscribing to topic: " + err.Error())
				}
			}
		} else if entry.Source.HttpSource != nil {
			d.log("Entry for http: " + entry.Name)
			for _, urlDef := range entry.Source.HttpSource.Urls {
				go func(e types.Entry, u string) {
					ticker := time.NewTicker(time.Duration(entry.Source.HttpSource.IntervalSec) * time.Second)
					defer ticker.Stop()
					d.log("- Polling " + u + " with interval: " + time.Duration(entry.Source.HttpSource.IntervalSec).String())

					tickFunc := func(url string, entry types.Entry) {
						payload, err := getHttpPayload(url)
						if err != nil {
							d.log("Error getting HTTP payload: " + err.Error())
							return
						}
						for _, topicPub := range entry.TopicsToPublish {
							d.callback(payload, entry, url, urlDef, topicPub, topicPub, func(msg []byte) {
								d.mqttClient.Publish(topicPub.Topic, msg)
							})
						}
					}

					tickFunc(u, e) // First tick
					for range ticker.C {
						tickFunc(u, e)
					}

				}(entry, urlDef.Url)
			}
		}
	}
}

func getHttpPayload(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

// callback is called when a new event is received
//
// 1. Tranform the payload to a float64, use types.TransformDefinition
// 2. If Accumulated, store the state in the dispatcher, and calculate the new value (use 'func (e Entry) MustAccumulate()')
// 3. Create a formatted message with OutputFormat
// 4. Create a message for TopicsToPublish with the formatted message and optional icon and color
func (d *Dispatcher) callback(payload []byte, entry types.Entry, id string, t types.Transform, o types.OuputFormat, f types.Filter, publish func([]byte)) {
	// 1) Transform payload
	val, err := d.transformPayload(payload, t)
	if err != nil {
		d.log("transform error: " + err.Error())
		return
	}

	// 2) Check if accumulation is needed
	if must, op := entry.MustAccumulate(); must {
		// Store to state
		if _, ok := d.state[entry.Name]; !ok {
			d.state[entry.Name] = make(map[string]float64)
		}
		d.state[entry.Name][id] = val

		// Calculate new value
		if op == types.Sum {
			sum := 0.0
			for _, v := range d.state[entry.Name] {
				sum += v
			}
			val = sum
			d.log(fmt.Sprintf("Accumulated value for %s: %f, from %v values ", entry.Name, val, len(d.state[entry.Name])))
		}
	}

	// Check for Filter
	if f.GetFilter() != nil {
		if f.GetFilter().IgnoreLessThan != nil {
			if val < *f.GetFilter().IgnoreLessThan {
				// Empty payload deletes the "custom app" on the client
				publish([]byte{})
				return
			}
		}
	}

	// 3) Format payload with optional color/icon
	pubMsg := publishMessage{}

	// Output Format
	formatted := outputFormat(val, o)
	pubMsg.Text = formatted

	// Color
	if entry.ColorScriptCallback != nil {
		if c, err := entry.ColorScriptCallback(val); err == nil {
			pubMsg.Color = c
		}
	}

	// Icon
	if entry.Icon != "" {
		pubMsg.Icon = entry.Icon
	}

	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		d.log(fmt.Sprintf("Error marshaling json: %v", err))
		publish([]byte(`{"text": "ERR"}`))
	}

	publish(jsonData)
}

func outputFormat(val float64, o types.OuputFormat) string {
	if o.GetOutputFormat() != "" {
		return fmt.Sprintf(o.GetOutputFormat(), val)
	}
	return fmt.Sprintf("%v", val)
}

func (d *Dispatcher) transformPayload(payload []byte, t types.Transform) (float64, error) {
	jsonPath := t.GetJsonPath()
	result := 0.0

	var err error
	if jsonPath == "" {
		result, err = strconv.ParseFloat(fmt.Sprintf("%v", string(payload)), 64)
		if err != nil {
			return 0, err
		}
	} else {
		var json_data interface{}
		json.Unmarshal([]byte(payload), &json_data)

		var err error
		var res interface{}
		res, err = jsonpath.JsonPathLookup(json_data, jsonPath)
		if err != nil {
			d.log(fmt.Sprintf("transformPayload JsonPath error: %v input: %s jsonPath: %s", err, string(payload), jsonPath))
			return 0, err
		}

		result, err = strconv.ParseFloat(fmt.Sprintf("%v", res), 64)
		if err != nil {
			d.log(fmt.Sprintf("transformPayload ParseError: %v input: %s jsonPath: %s", err, string(payload), jsonPath))
			return 0, err
		}
	}

	// Invert
	if t.GetInvert() {
		result = -result
	}

	return result, nil
}
