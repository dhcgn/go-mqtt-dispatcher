package dispatcher

import (
	"encoding/json"
	"fmt"
	"go-mqtt-dispatcher/config"
	httpsimple "go-mqtt-dispatcher/dispatcher/httpsimple"
	tibberapi "go-mqtt-dispatcher/dispatcher/tibber-api"
	tibbergraph "go-mqtt-dispatcher/tibber-graph"
	"go-mqtt-dispatcher/utils"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/oliveagle/jsonpath"
)

type dispatcherState map[string]map[string]float64
type Dispatcher struct {
	entries    *[]config.Entry
	state      dispatcherState
	mqttClient MqttClient
	log        func(string)
}

func NewDispatcher(entries *[]config.Entry, mqttClient MqttClient, log func(s string)) (*Dispatcher, error) {
	if log == nil {
		log = func(s string) {}
	}

	return &Dispatcher{
		entries:    entries,
		state:      make(dispatcherState),
		mqttClient: mqttClient,
		log:        log,
	}, nil
}

// Run starts the dispatcher and creates triggers for the sources and attaches the callbacks
func (d *Dispatcher) Run() {
	for _, entry := range *d.entries {
		if entry.Disabled {
			d.log("Entry disabled: " + entry.Name)
			continue
		}

		if entry.Source.MqttSource != nil {
			mqttEntry := config.MqttEntryImpl{Entry: entry}
			d.runMqtt(mqttEntry)
		} else if entry.Source.HttpSource != nil {
			httpEntry := config.HttpEntryImpl{Entry: entry}
			d.runHttp(httpEntry)
		} else if entry.Source.TibberApiSource != nil {
			tibberApiEntry := config.TibberApiEntryImpl{Entry: entry}
			d.runTibberApi(tibberApiEntry)
		}
	}
}

var (
	getTicker = func(d time.Duration) *time.Ticker {
		return time.NewTicker(d)
	}
)

func (d *Dispatcher) runTibberApi(entry config.TibberApiEntry) {
	d.log("Entry for " + entry.GetTypeName() + ": " + entry.GetName() + " with ID: " + entry.GetID())
	go func(e config.TibberApiEntry) {
		entry := e

		ticker := getTicker(time.Duration(entry.GetTibberApiSource().IntervalSec) * time.Second)
		defer ticker.Stop()
		d.log("- Polling from tibber API with interval: " + time.Duration(entry.GetTibberApiSource().IntervalSec*int(time.Second)).String())

		tickFunc := func(entry config.TibberApiEntry) {
			payload, err := tibberapi.GetTibberAPIPayload(entry.GetTibberApiSource().TibberApiKey, entry.GetTibberApiSource().GraphqlQuery)
			if err != nil {
				d.log("Error getting HTTP payload: " + err.Error())
				return
			}
			for _, topicPub := range entry.GetTopicsToPublish() {
				c := callbackConfig{Entry: e.GetEntry(), Id: entry.GetID(), TransSource: entry.GetTibberApiSource(), TransTarget: topicPub, Filter: topicPub}
				d.callback(payload, c, func(msg []byte) {
					d.mqttClient.Publish(topicPub.Topic, msg)
				})
			}
		}

		tickFunc(entry) // First tick
		for range ticker.C {
			tickFunc(entry)
			if interruptRunHttpTickerAfterTick {
				return
			}
		}

	}(entry)
}

// runHttp creates a trigger for the http source and attaches the callback
func (d *Dispatcher) runHttp(entry config.HttpEntry) {
	d.log("Entry for " + entry.GetTypeName() + ": " + entry.GetName() + " with ID: " + entry.GetID())
	for _, urlDef := range entry.GetSources() {
		go func(e config.HttpEntry, u string) {
			tickerduration := time.Duration(time.Duration(entry.GetIntervalSec()) * time.Second)
			ticker := getTicker(tickerduration)
			defer ticker.Stop()
			d.log("- Polling " + u + " with interval: " + tickerduration.String())

			tickFunc := func(url string, entry config.HttpEntry) {
				payload, err := httpsimple.GetHttpPayload(url)
				if err != nil {
					d.log("Error getting HTTP payload: " + err.Error())
					return
				}
				for _, topicPub := range entry.GetTopicsToPublish() {
					c := callbackConfig{Entry: entry.GetEntry(), Id: url, TransSource: urlDef, TransTarget: topicPub, Filter: topicPub}
					d.callback(payload, c, func(msg []byte) {
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
func (d *Dispatcher) runMqtt(entry config.MqttEntry) {
	d.log("Entry for " + entry.GetTypeName() + ": " + entry.GetName() + " with ID: " + entry.GetID())
	for _, topicSub := range entry.GetTopicsToSubscribe() {
		d.log("- Subscribing to " + topicSub.Topic)
		err := d.mqttClient.Subscribe(topicSub.Topic, func(payload []byte) {
			d.log("Received payload for " + topicSub.Topic)
			for _, topicPub := range entry.GetTopicsToPublish() {
				c := callbackConfig{Entry: entry.GetEntry(), Id: topicSub.Topic, TransSource: topicSub, TransTarget: topicPub, Filter: topicPub}
				d.callback(payload, c, func(msg []byte) {
					d.mqttClient.Publish(topicPub.Topic, msg)
				})
			}
		})
		if err != nil {
			d.log("Error subscribing to topic: " + err.Error())
		}
	}
}

type callbackConfig struct {
	Entry       config.Entry
	Id          string
	TransSource config.TransformSource
	TransTarget config.TransformTarget
	Filter      config.Filter
}

var errorPayload = []byte(`{"text": "ERR"}`)

func (d *Dispatcher) getOutputAsTibberGraph(payload []byte, c config.TransformSource) ([]byte, error) {

	payload = utils.TransformPayloadWithJsonPath(payload, c)

	t := time.Now()
	g, err := tibbergraph.CreateDraw(string(payload), t)
	if err != nil {
		d.log("Error creating graph: " + err.Error())
		return nil, err
	}
	j, err := g.GetJson()
	if err != nil {
		d.log("Error getting json: " + err.Error())
		return nil, err
	}

	d.log("Generated TibberGraph with " + strconv.Itoa(len(g.Draw)) + " rows and with time: " + t.String())

	return []byte(j), nil
}

// callback is called when a new event is received
func (d *Dispatcher) callback(payload []byte, c callbackConfig, publish func([]byte)) {
	if c.TransTarget != nil && c.TransTarget.GetOutputAsTibberGraph() {
		p, err := d.getOutputAsTibberGraph(payload, c.TransSource)
		if err != nil {
			d.log("Error getting TibberGraph: " + err.Error())
			publish(errorPayload)
			return
		}
		publish(p)
		return
	}

	val, err := d.transformPayload(payload, c.TransSource)
	if err != nil {
		d.log("transform error: " + err.Error())
		return
	}

	// Accumulate
	if must, op := c.Entry.MustAccumulate(); must {
		if _, ok := d.state[c.Entry.Name]; !ok {
			d.state[c.Entry.Name] = make(map[string]float64)
		}
		d.state[c.Entry.Name][c.Id] = val

		if op == config.OperatorSum {
			sum := 0.0
			for _, v := range d.state[c.Entry.Name] {
				sum += v
			}
			val = sum
			d.log(fmt.Sprintf("Accumulated value for %s: %f, from %v values ", c.Entry.Name, val, len(d.state[c.Entry.Name])))
		} else {
			d.log(fmt.Sprintf("Operation '%s' not supported", op))
		}
	}

	// Filter
	if c.Filter.GetFilter() != nil {
		if c.Filter.GetFilter().IgnoreLessThan != nil {
			if val < *c.Filter.GetFilter().IgnoreLessThan {
				// Empty payload deletes the "custom app" on the client
				publish([]byte{})
				return
			}
		}
	}

	pubMsg := publishMessage{}

	// Output Format
	formatted := outputFormat(val, c.TransTarget)
	pubMsg.Text = formatted

	// Add Color
	if c.Entry.ColorScriptCallback != nil {
		if c, err := c.Entry.ColorScriptCallback(val); err == nil {
			pubMsg.Color = c
		}
	}

	// Add Icon
	if c.Entry.Icon != "" {
		pubMsg.Icon = c.Entry.Icon
	}

	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		d.log(fmt.Sprintf("Error marshaling json: %v", err))
		publish(errorPayload)
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
