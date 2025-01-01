package dispatcher

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
)

type publishMessage struct {
	Text  string `json:"text"`
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

func createColorCallback(script string) (func(float64) (string, error), error) {
	if script == "" {
		return nil, errors.New("Empty script")
	}

	vm := goja.New()
	_, err := vm.RunString(script)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error running color script: %v", err))
	}

	get_color, ok := goja.AssertFunction(vm.Get("get_color"))
	if !ok {
		return nil, errors.New("Error getting get_color function from script")
	}
	res, err := get_color(goja.Undefined(), vm.ToValue(1.0))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error running color script: %v", err))
	}

	hexcode := res.String()
	if len(hexcode) != 7 || hexcode[0] != '#' {
		return nil, errors.New("Invalid hex color code")
	}

	return func(value float64) (string, error) {
		res, err := get_color(goja.Undefined(), vm.ToValue(value))
		if err != nil {
			return "", err
		}
		return res.String(), nil
	}, nil
}
