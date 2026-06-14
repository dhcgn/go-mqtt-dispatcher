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
	"sync"
	"time"
	"unicode"

	"github.com/oliveagle/jsonpath"
)

// now is the clock used by the fallback feature. It is a package var so tests
// can override it for deterministic time (mirrors getTicker).
var now = func() time.Time { return time.Now() }

type dispatcherState map[string]map[string]float64

// fallbackTrack records freshness state for one publish topic of an entry with
// a stale-value fallback configured.
type fallbackTrack struct {
	lastActivity time.Time // last time any payload was received (no-value-read)
	lastValue    float64   // last resolved value (no-value-change)
	hasValue     bool      // lastValue is valid
	lastChange   time.Time // last time the resolved value changed (no-value-change)
	fired        bool      // fallback already published; suppresses re-fire until fresh data
}

type Dispatcher struct {
	entries    *[]config.Entry
	state      dispatcherState
	mqttClient MqttClient
	log        func(string)

	// mu protects fallbacks only. The pre-existing `state` map is intentionally
	// left unsynchronized (unchanged behavior).
	mu        sync.Mutex
	fallbacks map[string]*fallbackTrack // key = fallbackKey(entry.Name, pubTopic)
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
		fallbacks:  make(map[string]*fallbackTrack),
	}, nil
}

func fallbackKey(entryName, pubTopic string) string {
	return entryName + "\x00" + pubTopic
}

// Run starts the dispatcher and creates triggers for the sources and attaches the callbacks
func (d *Dispatcher) Run() {
	for _, entry := range *d.entries {
		if entry.Disabled {
			d.log("Entry disabled: " + entry.Name)
			continue
		}

		if entry.HasFallback() {
			d.startFallbackWatchdog(entry)
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
				c := callbackConfig{Entry: e.GetEntry(), Id: entry.GetID(), PubTopic: topicPub.Topic, TransSource: entry.GetTibberApiSource(), TransTarget: topicPub, Filter: topicPub}
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
					c := callbackConfig{Entry: entry.GetEntry(), Id: url, PubTopic: topicPub.Topic, TransSource: urlDef, TransTarget: topicPub, Filter: topicPub}
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
				c := callbackConfig{Entry: entry.GetEntry(), Id: topicSub.Topic, PubTopic: topicPub.Topic, TransSource: topicSub, TransTarget: topicPub, Filter: topicPub}
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
	PubTopic    string
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
	// Any received payload counts as activity for the no-value-read fallback,
	// including the tibber-graph, filter, and transform-error paths below.
	if c.Entry.HasFallback() {
		d.markActivity(c.Entry.Name, c.PubTopic)
	}

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

	// Track the resolved (possibly accumulated) value for no-value-change.
	if c.Entry.HasFallback() {
		d.markValue(c.Entry.Name, c.PubTopic, val)
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

// markActivity records that a payload was received for a publish topic. Used by
// the no-value-read fallback. No-op when no track exists (fallback disabled).
func (d *Dispatcher) markActivity(entryName, pubTopic string) {
	if entryName == "" || pubTopic == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.fallbacks[fallbackKey(entryName, pubTopic)]
	if t == nil {
		return
	}
	t.lastActivity = now()
	t.fired = false
}

// markValue records the resolved value for a publish topic. Used by the
// no-value-change fallback; resets the fallback only when the value changes.
func (d *Dispatcher) markValue(entryName, pubTopic string, val float64) {
	if entryName == "" || pubTopic == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.fallbacks[fallbackKey(entryName, pubTopic)]
	if t == nil {
		return
	}
	n := now()
	t.lastActivity = n
	if !t.hasValue || t.lastValue != val {
		t.lastValue = val
		t.hasValue = true
		t.lastChange = n
		t.fired = false
	}
}

// seedFallback initializes tracking state for each publish topic of an entry so
// markActivity/markValue have something to update and a never-delivering source
// can still go stale (relative to startup).
func (d *Dispatcher) seedFallback(entry config.Entry) {
	start := now()
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, pub := range entry.TopicsToPublish {
		k := fallbackKey(entry.Name, pub.Topic)
		if d.fallbacks[k] == nil {
			d.fallbacks[k] = &fallbackTrack{lastActivity: start, lastChange: start}
		}
	}
}

// fireFallbacksIfStale publishes the fallback payload to any of the entry's
// publish topics that have gone stale, marking each as fired so it is published
// at most once per stale episode.
func (d *Dispatcher) fireFallbacksIfStale(entry config.Entry) {
	n := now()
	mode := entry.FallbackMode()
	after := entry.FallbackAfter

	var due []string
	d.mu.Lock()
	for _, pub := range entry.TopicsToPublish {
		t := d.fallbacks[fallbackKey(entry.Name, pub.Topic)]
		if t == nil || t.fired {
			continue
		}
		var stale bool
		switch mode {
		case config.FallbackModeNoValueRead:
			stale = n.Sub(t.lastActivity) >= after
		case config.FallbackModeNoValueChange:
			stale = t.hasValue && n.Sub(t.lastChange) >= after
		}
		if stale {
			t.fired = true
			due = append(due, pub.Topic)
		}
	}
	d.mu.Unlock()

	// Publish outside the lock.
	if len(due) == 0 {
		return
	}
	payload := d.fallbackPayload(entry)
	for _, topic := range due {
		d.log("Fallback firing for " + entry.Name + " -> " + topic)
		d.mqttClient.Publish(topic, payload)
	}
}

// startFallbackWatchdog seeds tracking state and starts a goroutine that
// periodically publishes the configured fallback once a topic goes stale.
func (d *Dispatcher) startFallbackWatchdog(entry config.Entry) {
	d.seedFallback(entry)

	interval := entry.FallbackAfter / 4
	if interval < time.Second {
		interval = time.Second
	}

	d.log(fmt.Sprintf("- Fallback watchdog for %s: mode=%s after=%s", entry.Name, entry.FallbackMode(), entry.FallbackAfter))

	go func(entry config.Entry) {
		ticker := getTicker(interval)
		defer ticker.Stop()

		d.fireFallbacksIfStale(entry)
		for range ticker.C {
			d.fireFallbacksIfStale(entry)
			if interruptRunHttpTickerAfterTick {
				return
			}
		}
	}(entry)
}

// fallbackPayload builds the literal fallback message for an entry. The value
// and color are literal (they bypass outputFormat and the color-script); the
// entry's icon is reused.
func (d *Dispatcher) fallbackPayload(entry config.Entry) []byte {
	msg := publishMessage{
		Text:  entry.Fallback.Value,
		Color: entry.Fallback.Color,
		Icon:  entry.Icon,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		d.log(fmt.Sprintf("Error marshaling fallback json: %v", err))
		return errorPayload
	}
	return b
}
