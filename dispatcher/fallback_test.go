package dispatcher

import (
	"go-mqtt-dispatcher/config"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fbClock backs an injectable, race-free test clock for the fallback feature.
var fbClock atomic.Int64

func useTestClock(base time.Time) {
	fbClock.Store(base.UnixNano())
	now = func() time.Time { return time.Unix(0, fbClock.Load()) }
}

func setClock(t time.Time) { fbClock.Store(t.UnixNano()) }

// lastMessage returns the last payload published to a topic as a string.
func lastMessage(mc *MockMqttClient, topic string) string {
	msg, _ := mc.GetPublishedMessage(topic)
	return string(msg)
}

// fallbackPayloadJSON is the expected payload for the test entries below.
const fallbackPayloadJSON = `{"text":"? °C","icon":"pool","color":"#888888"}`

func newFallbackEntry(mode string, pubTopics ...string) config.Entry {
	pubs := make([]config.MqttTopicDefinition, 0, len(pubTopics))
	for _, p := range pubTopics {
		pubs = append(pubs, config.MqttTopicDefinition{Topic: p, Transform: config.TransformDefinition{OutputFormat: "%.1f C"}})
	}
	return config.Entry{
		Name: "tempEntry",
		Icon: "pool",
		Source: config.EntrySource{
			MqttSource: &config.MqttSource{
				TopicsToSubscribe: []config.MqttTopicDefinition{
					{Topic: "sensor/temp", Transform: config.TransformDefinition{JsonPath: "$.t"}},
				},
			},
		},
		TopicsToPublish: pubs,
		Fallback:        &config.FallbackDefinition{Mode: mode, After: "1h", Value: "? °C", Color: "#888888"},
		FallbackAfter:   time.Hour,
	}
}

// setup builds a dispatcher, seeds fallback tracking, and wires the mqtt
// subscription so SimulateMessage drives the real callback path.
func setup(t *testing.T, entry config.Entry) (*Dispatcher, *MockMqttClient) {
	t.Helper()
	log := func(s string) { t.Log(s) }
	mc := NewMockMqttClient(log)
	d, err := NewDispatcher(&[]config.Entry{entry}, mc, log)
	require.NoError(t, err)
	if entry.HasFallback() {
		d.seedFallback(entry)
	}
	d.runMqtt(config.MqttEntryImpl{Entry: entry})
	return d, mc
}

func TestFallbackNoValueReadFires(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := newFallbackEntry("no-value-read", "display/temp")
	d, mc := setup(t, entry)

	// Not yet stale.
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, 0, mc.GetPublishCount("display/temp"))

	// Past the timeout with no data -> fires once.
	setClock(base.Add(2 * time.Hour))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, 1, mc.GetPublishCount("display/temp"))
	assert.Equal(t, fallbackPayloadJSON, lastMessage(mc, "display/temp"))

	// Fire-once: subsequent checks do not republish.
	d.fireFallbacksIfStale(entry)
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, 1, mc.GetPublishCount("display/temp"))
}

func TestFallbackNoValueReadResetByMessage(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := newFallbackEntry("no-value-read", "display/temp")
	d, mc := setup(t, entry)

	// A message arrives before the timeout, resetting the activity clock.
	setClock(base.Add(30 * time.Minute))
	mc.SimulateMessage("sensor/temp", []byte(`{"t":42}`))

	// 1h after startup but only 31m after the message -> not stale.
	setClock(base.Add(1*time.Hour + 1*time.Minute))
	before := mc.GetPublishCount("display/temp")
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, before, mc.GetPublishCount("display/temp"))

	// Past 1h after the message -> fires.
	setClock(base.Add(1*time.Hour + 31*time.Minute))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, before+1, mc.GetPublishCount("display/temp"))
	assert.Equal(t, fallbackPayloadJSON, lastMessage(mc, "display/temp"))
}

func TestFallbackNoValueChangeFires(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := newFallbackEntry("no-value-change", "display/temp")
	d, mc := setup(t, entry)

	// Before any value, change-mode must not fire even past the timeout.
	setClock(base.Add(2 * time.Hour))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, 0, mc.GetPublishCount("display/temp"))

	// First value seen.
	mc.SimulateMessage("sensor/temp", []byte(`{"t":42}`))
	before := mc.GetPublishCount("display/temp")

	// Same value, well past the timeout -> fires.
	setClock(base.Add(4 * time.Hour))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, before+1, mc.GetPublishCount("display/temp"))
	assert.Equal(t, fallbackPayloadJSON, lastMessage(mc, "display/temp"))
}

func TestFallbackNoValueChangeResetByChange(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := newFallbackEntry("no-value-change", "display/temp")
	d, mc := setup(t, entry)

	mc.SimulateMessage("sensor/temp", []byte(`{"t":42}`))

	// A changed value before the timeout resets the change clock.
	setClock(base.Add(50 * time.Minute))
	mc.SimulateMessage("sensor/temp", []byte(`{"t":43}`))
	before := mc.GetPublishCount("display/temp")

	// Only 30m since the change -> not stale.
	setClock(base.Add(1*time.Hour + 20*time.Minute))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, before, mc.GetPublishCount("display/temp"))

	// Past 1h since the change -> fires.
	setClock(base.Add(1*time.Hour + 51*time.Minute))
	d.fireFallbacksIfStale(entry)
	assert.Equal(t, before+1, mc.GetPublishCount("display/temp"))
}

func TestFallbackDisabled(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := config.Entry{
		Name: "noFallback",
		Source: config.EntrySource{
			MqttSource: &config.MqttSource{
				TopicsToSubscribe: []config.MqttTopicDefinition{
					{Topic: "sensor/temp", Transform: config.TransformDefinition{JsonPath: "$.t"}},
				},
			},
		},
		TopicsToPublish: []config.MqttTopicDefinition{{Topic: "display/temp"}},
	}
	require.False(t, entry.HasFallback())

	d, mc := setup(t, entry)

	// No tracking was seeded; a stale check never publishes a fallback.
	setClock(base.Add(100 * time.Hour))
	d.fireFallbacksIfStale(entry)
	_, ok := mc.GetPublishedMessage("display/temp")
	assert.False(t, ok)
	assert.Empty(t, d.fallbacks)
}

func TestFallbackMultipleTopics(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	useTestClock(base)

	entry := newFallbackEntry("no-value-read", "display/a", "display/b")
	d, mc := setup(t, entry)

	setClock(base.Add(2 * time.Hour))
	d.fireFallbacksIfStale(entry)

	assert.Equal(t, 1, mc.GetPublishCount("display/a"))
	assert.Equal(t, 1, mc.GetPublishCount("display/b"))
	assert.Equal(t, fallbackPayloadJSON, lastMessage(mc, "display/a"))
	assert.Equal(t, fallbackPayloadJSON, lastMessage(mc, "display/b"))
}
