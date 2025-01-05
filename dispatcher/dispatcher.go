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
	"go.uber.org/zap"
)

type dispatcherState map[string]map[string]float64
type Dispatcher struct {
	entries    *[]config.Entry
	state      dispatcherState
	mqttClient MqttClient
	logger     *zap.SugaredLogger
}

func NewDispatcher(entries *[]config.Entry, mqttClient MqttClient, logger *zap.SugaredLogger) (*Dispatcher, error) {

	return &Dispatcher{
		entries:    entries,
		state:      make(dispatcherState),
		mqttClient: mqttClient,
		logger:     logger,
	}, nil
}

// Run starts the dispatcher and creates triggers for the sources and attaches the callbacks
func (d *Dispatcher) Run() {
	for _, entry := range *d.entries {
		if entry.Disabled {
			d.logger.Infow("Entry disabled", "name", entry.Name, "id", entry.GetID)
			continue
		}

		if entry.Source.MqttSource != nil {
			mqttEntry := config.MqttEntryImpl{Entry: entry}
			d.runMqtt(mqttEntry, d.logger.Named("mqtt"))
		} else if entry.Source.HttpSource != nil {
			httpEntry := config.HttpEntryImpl{Entry: entry}
			d.runHttp(httpEntry, d.logger.Named("http"))
		} else if entry.Source.TibberApiSource != nil {
			tibberApiEntry := config.TibberApiEntryImpl{Entry: entry}
			d.runTibberApi(tibberApiEntry, d.logger.Named("tibber-api"))
		}
	}
}

var (
	getTicker = func(d time.Duration) *time.Ticker {
		return time.NewTicker(d)
	}
)

func (d *Dispatcher) runTibberApi(entry config.TibberApiEntry, logger *zap.SugaredLogger) {
	logger.Infow("Entry for Tibber API", "type", entry.GetTypeName(), "name", entry.GetName(), "id", entry.GetID())
	go func(e config.TibberApiEntry) {
		entry := e

		ticker := getTicker(time.Duration(entry.GetTibberApiSource().IntervalSec) * time.Second)
		defer ticker.Stop()
		logger.Infow("Polling from Tibber API", "interval", time.Duration(entry.GetTibberApiSource().IntervalSec*int(time.Second)).String())

		tickFunc := func(entry config.TibberApiEntry) {
			payload, err := tibberapi.GetTibberAPIPayload(entry.GetTibberApiSource().TibberApiKey, entry.GetTibberApiSource().GraphqlQuery)
			if err != nil {
				logger.Errorw("Error getting HTTP payload", "error", err)
				return
			}
			for _, topicPub := range entry.GetTopicsToPublish() {
				c := callbackConfig{Entry: e.GetEntry(), Id: entry.GetID(), TransSource: entry.GetTibberApiSource(), TransTarget: topicPub, Filter: topicPub, Logger: logger.Named("callback")}
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
func (d *Dispatcher) runHttp(entry config.HttpEntry, logger *zap.SugaredLogger) {
	logger.Infow("Entry for HTTP", "type", entry.GetTypeName(), "name", entry.GetName(), "id", entry.GetID())
	for _, urlDef := range entry.GetSources() {
		go func(e config.HttpEntry, u string) {
			tickerduration := time.Duration(time.Duration(entry.GetIntervalSec()) * time.Second)
			ticker := getTicker(tickerduration)
			defer ticker.Stop()
			logger.Infow("Polling URL", "url", u, "interval", tickerduration.String())

			tickFunc := func(url string, entry config.HttpEntry) {
				payload, err := httpsimple.GetHttpPayload(url)
				if err != nil {
					logger.Errorw("Error getting HTTP payload", "error", err)
					return
				}
				for _, topicPub := range entry.GetTopicsToPublish() {
					c := callbackConfig{Entry: entry.GetEntry(), Id: url, TransSource: urlDef, TransTarget: topicPub, Filter: topicPub, Logger: logger.Named("callback")}
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
func (d *Dispatcher) runMqtt(entry config.MqttEntry, logger *zap.SugaredLogger) {
	logger.Infow("Entry for MQTT", "type", entry.GetTypeName(), "name", entry.GetName(), "id", entry.GetID())
	for _, topicSub := range entry.GetTopicsToSubscribe() {
		logger.Infow("Subscribing to topic", "topic", topicSub.Topic)
		err := d.mqttClient.Subscribe(topicSub.Topic, func(payload []byte) {
			logger.Infow("Received payload for topic", "topic", topicSub.Topic)
			for _, topicPub := range entry.GetTopicsToPublish() {
				c := callbackConfig{Entry: entry.GetEntry(), Id: topicSub.Topic, TransSource: topicSub, TransTarget: topicPub, Filter: topicPub, Logger: logger.Named("callback")}
				d.callback(payload, c, func(msg []byte) {
					d.mqttClient.Publish(topicPub.Topic, msg)
				})
			}
		})
		if err != nil {
			logger.Errorw("Error subscribing to topic", "error", err)
		}
	}
}

type callbackConfig struct {
	Entry       config.Entry
	Id          string
	TransSource config.TransformSource
	TransTarget config.TransformTarget
	Filter      config.Filter
	Logger      *zap.SugaredLogger // Add this field
}

var errorPayload = []byte(`{"text": "ERR"}`)

func (d *Dispatcher) getOutputAsTibberGraph(payload []byte, c config.TransformSource) ([]byte, error) {

	payload = utils.TransformPayloadWithJsonPath(payload, c)

	t := time.Now()
	g, err := tibbergraph.CreateDraw(string(payload), t)
	if err != nil {
		d.logger.Errorw("Error creating graph", "error", err)
		return nil, err
	}
	j, err := g.GetJson()
	if err != nil {
		d.logger.Errorw("Error getting JSON", "error", err)
		return nil, err
	}

	d.logger.Infow("Generated TibberGraph", "rows", len(g.Draw), "time", t.String())

	return []byte(j), nil
}

// callback is called when a new event is received
func (d *Dispatcher) callback(payload []byte, c callbackConfig, publish func([]byte)) {
	if c.TransTarget != nil && c.TransTarget.GetOutputAsTibberGraph() {
		p, err := d.getOutputAsTibberGraph(payload, c.TransSource)
		if err != nil {
			c.Logger.Errorw("Error getting TibberGraph", "error", err)
			publish(errorPayload)
			return
		}
		publish(p)
		return
	}

	val, err := d.transformPayload(payload, c.TransSource)
	if err != nil {
		c.Logger.Errorw("Transform error", "error", err)
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
			c.Logger.Infow("Accumulated value", "entry", c.Entry.Name, "value", val, "count", len(d.state[c.Entry.Name]))
		} else {
			c.Logger.Warnw("Operation not supported", "operation", op)
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
		if color, err := c.Entry.ColorScriptCallback(val); err == nil {
			pubMsg.Color = color
		} else {
			c.Logger.Errorw("Error getting color", "error", err)
		}
	}

	// Add Icon
	if c.Entry.Icon != "" {
		pubMsg.Icon = c.Entry.Icon
	}

	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		c.Logger.Errorw("Error marshaling JSON", "error", err)
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
			d.logger.Errorw("JsonPath error", "error", err, "input", string(payload), "jsonPath", jsonPath)
			return 0, err
		}

		result, err = strconv.ParseFloat(fmt.Sprintf("%v", res), 64)
		if err != nil {
			d.logger.Errorw("Parse error", "error", err, "input", string(payload), "jsonPath", jsonPath)
			return 0, err
		}
	}

	// Invert
	if t.GetInvert() {
		result = -result
	}

	return result, nil
}
