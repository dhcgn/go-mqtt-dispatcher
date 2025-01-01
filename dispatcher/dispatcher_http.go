package dispatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go-mqtt-dispatcher/types"

	"github.com/oliveagle/jsonpath"
)

type DispatcherHttp struct {
	entries    *[]types.HttpEntry
	mqttClient MqttClient
	log        func(string)
}

func NewDispatcherHttp(httpEntries *[]types.HttpEntry, mqttClient MqttClient, log func(s string)) (*DispatcherHttp, error) {
	if log == nil {
		log = func(s string) {}
	}

	d := &DispatcherHttp{
		entries:    httpEntries,
		mqttClient: mqttClient,
		log:        log,
	}

	// Check ColorScript in each types.HttpConfig an set callback
	for cfg_i, cfg := range *httpEntries {
		if cfg.ColorScript != "" {
			colorCallback, err := createColorCallback(cfg.ColorScript)
			if err != nil {
				log(fmt.Sprintf("Error creating color callback for config %d: %v", cfg_i, err))
			}
			(*d.entries)[cfg_i].ColorScriptCallback = colorCallback
		}
	}

	return d, nil
}

func extractToFloatHTTP(body []byte, transform types.TransformConfig) (float64, error) {
	// If no JSON path, assume the entire body is a numeric value
	if transform.JsonPath == "" {
		val, err := strconv.ParseFloat(string(body), 64)
		if err != nil {
			return 0, err
		}
		if transform.Invert {
			val = -val
		}
		return val, nil
	}

	// Otherwise, unmarshal and extract using JSON path
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return 0, err
	}
	result, err := jsonpath.JsonPathLookup(jsonData, transform.JsonPath)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseFloat(fmt.Sprintf("%v", result), 64)
	if err != nil {
		return 0, err
	}
	if transform.Invert {
		val = -val
	}
	return val, nil
}

func (d *DispatcherHttp) Run() {
	for _, cfg := range *d.entries {
		cfg := cfg
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.IntervalSec) * time.Second)
			defer ticker.Stop()

			d.log("Starting HTTP dispatcher for: " + cfg.Url + " -> " + cfg.Publish + " with interval: " + strconv.Itoa(cfg.IntervalSec) + "s")
			d.tick(cfg) // First tick

			for range ticker.C {
				d.tick(cfg)
			}
		}()
	}
}

func (d *DispatcherHttp) tick(cfg types.HttpConfig) {
	resp, err := http.Get(cfg.Url)
	if err != nil {
		d.log("Failed to fetch: " + cfg.Url)
		return
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		d.log("Failed to read response body")
		return
	}

	val, err := extractToFloatHTTP(body, cfg.Transform)
	if err != nil {
		d.log("HTTP JSONPath error: " + err.Error())
		return
	}

	// Format final output
	var output string
	if cfg.Transform.OutputFormat != "" {
		output = fmt.Sprintf(cfg.Transform.OutputFormat, val)
	} else {
		output = fmt.Sprintf("%v", val)
	}

	colorHex := ""
	if cfg.ColorScriptCallback != nil {
		c, err := cfg.ColorScriptCallback(val)
		if err != nil {
			d.log(fmt.Sprintf("Error running color script: %v", err))
		} else {
			colorHex = c
		}
	}

	pubMsg := publishMessage{
		Text:  output,
		Icon:  cfg.Icon,
		Color: colorHex,
	}
	jsonData, err := json.Marshal(pubMsg)
	if err != nil {
		d.log(fmt.Sprintf("Error marshaling json: %v", err))
		return
	}

	d.mqttClient.Publish(cfg.Publish, jsonData)
}
