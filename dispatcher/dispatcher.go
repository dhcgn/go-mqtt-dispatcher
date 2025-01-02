package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/config"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/oliveagle/jsonpath"
)

type Dispatcher struct {
	entries    *[]config.Entry
	state      map[string]map[string]float64
	mqttClient MqttClient
	log        func(string)
}

func NewDispatcher(entries *[]config.Entry, mqttClient MqttClient, log func(s string)) (*Dispatcher, error) {
	if log == nil {
		log = func(s string) {}
	}

	return &Dispatcher{
		entries:    entries,
		state:      make(map[string]map[string]float64),
		mqttClient: mqttClient,
		log:        log,
	}, nil
}

// Run starts the dispatcher and creates triggers for the sources and attaches the callbacks
func (d *Dispatcher) Run() {
	for _, entry := range *d.entries {
		if entry.Source.MqttSource != nil {
			d.log("Entry for mqtt: " + entry.Name)
			d.runMqtt(entry)
		} else if entry.Source.HttpSource != nil {
			d.log("Entry for http: " + entry.Name)
			d.runHttp(entry)
		}
	}
}

// runHttp creates a trigger for the http source and attaches the callback
func (d *Dispatcher) runHttp(entry config.Entry) {
	for _, urlDef := range entry.Source.HttpSource.Urls {
		go func(e config.Entry, u string) {
			ticker := time.NewTicker(time.Duration(entry.Source.HttpSource.IntervalSec) * time.Second)
			defer ticker.Stop()
			d.log("- Polling " + u + " with interval: " + time.Duration(entry.Source.HttpSource.IntervalSec*int(time.Second)).String())

			tickFunc := func(url string, entry config.Entry) {
				payload, err := getHttpPayload(url)
				if err != nil {
					d.log("Error getting HTTP payload: " + err.Error())
					return
				}
				for _, topicPub := range entry.TopicsToPublish {
					d.callback(payload, entry, url, urlDef, topicPub, func(msg []byte) {
						d.mqttClient.Publish(topicPub.Topic, msg)
					})
				}
			}

			tickFunc(u, e) // First tick
			for range ticker.C {
				tickFunc(u, e)
				if interruptRunHttpTickerAfterTick {
					return
				}
			}

		}(entry, urlDef.Url)
	}
}

var (
	interruptRunHttpTickerAfterTick = false
)

// runMqtt creates a trigger for the mqtt source and attaches the callback
func (d *Dispatcher) runMqtt(entry config.Entry) {
	for _, topicSub := range entry.Source.MqttSource.TopicsToSubscribe {
		d.log("- Subscribing to " + topicSub.Topic)
		err := d.mqttClient.Subscribe(topicSub.Topic, func(payload []byte) {
			d.log("Received payload for " + topicSub.Topic)
			for _, topicPub := range entry.TopicsToPublish {
				d.callback(payload, entry, topicSub.Topic, topicSub, topicPub, func(msg []byte) {
					d.mqttClient.Publish(topicPub.Topic, msg)
				})
			}
		})
		if err != nil {
			d.log("Error subscribing to topic: " + err.Error())
		}
	}
}

var (
	httpGet = func(url string) (resp *http.Response, err error) {
		return http.Get(url)
	}
)

func getHttpPayload(url string) ([]byte, error) {
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

// callback is called when a new event is received
func (d *Dispatcher) callback(payload []byte, entry config.Entry, id string, trans config.Transformers, f config.Filter, publish func([]byte)) {
	val, err := d.transformPayload(payload, trans)
	if err != nil {
		d.log("transform error: " + err.Error())
		return
	}

	// Accumulate
	if must, op := entry.MustAccumulate(); must {
		if _, ok := d.state[entry.Name]; !ok {
			d.state[entry.Name] = make(map[string]float64)
		}
		d.state[entry.Name][id] = val

		if op == config.OperatorSum {
			sum := 0.0
			for _, v := range d.state[entry.Name] {
				sum += v
			}
			val = sum
			d.log(fmt.Sprintf("Accumulated value for %s: %f, from %v values ", entry.Name, val, len(d.state[entry.Name])))
		} else {
			d.log(fmt.Sprintf("Operation '%s' not supported", op))
		}
	}

	// Filter
	if f.GetFilter() != nil {
		if f.GetFilter().IgnoreLessThan != nil {
			if val < *f.GetFilter().IgnoreLessThan {
				// Empty payload deletes the "custom app" on the client
				publish([]byte{})
				return
			}
		}
	}

	pubMsg := publishMessage{}

	// Output Format
	formatted := outputFormat(val, trans)
	pubMsg.Text = formatted

	// Add Color
	if entry.ColorScriptCallback != nil {
		if c, err := entry.ColorScriptCallback(val); err == nil {
			pubMsg.Color = c
		}
	}

	// Add Icon
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

func outputFormat(val float64, o config.TransformTarget) string {
	if o.GetOutputFormat() != "" {
		return fmt.Sprintf(o.GetOutputFormat(), val)
	}
	return fmt.Sprintf("%v", val)
}

func (d *Dispatcher) transformPayload(payload []byte, t config.TransformSource) (float64, error) {
	jsonPath := t.GetJsonPath()
	result := 0.0

	var err error
	if jsonPath == "" {
		trimmed := strings.TrimFunc(string(payload), func(r rune) bool {
			return !unicode.IsPrint(r)
		})
		result, err = strconv.ParseFloat(trimmed, 64)
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
