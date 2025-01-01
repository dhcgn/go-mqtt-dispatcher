package dispatcher

import "go-mqtt-dispatcher/types"

type Dispatcher struct {
}

func NewDispatcher(entries *[]types.Entry, mqttClient MqttClient, log func(s string)) (*Dispatcher, error) {
	return &Dispatcher{}, nil
}

// Run starts the dispatcher
//
// 1. Iterate over all entries
// 2. Create a event listener for each type of MqttSource or HttpSource
// 3. Create a callback function for each event listener
func (d *Dispatcher) Run() {

}

// callback is called when a new event is received
//
// 1. Tranform the payload to a float64, use types.TransformDefinition
// 2. If Accumulated, store the state in the dispatcher, and calculate the new value (use 'func (e Entry) MustAccumulate()')
// 3. Create a formatted message with OutputFormat
// 4. Create a message for TopicsToPublish with the formatted message and optional icon and color
func (d *Dispatcher) callback(payload []byte, entry types.Entry, publish func(string, []byte)) {

}
