package utils

import (
	"encoding/json"
	"go-mqtt-dispatcher/config"

	"github.com/oliveagle/jsonpath"
)

func TransformPayloadWithJsonPath(payload []byte, c config.TransformSource) []byte {
	if c.GetJsonPath() == "" {
		return payload
	}

	var json_data interface{}
	json.Unmarshal([]byte(payload), &json_data)

	res, err := jsonpath.JsonPathLookup(json_data, c.GetJsonPath())
	if err != nil {
		return []byte{}
	}

	j, err := json.Marshal(res)
	if err != nil {
		return []byte{}
	}

	return j
}
