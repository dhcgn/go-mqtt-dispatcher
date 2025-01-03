package config

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
)

const (
	errorEmptyScript   = "EMPTY SCRIPT"
	errorRunningScript = "ERROR RUNNING SCRIPT"
	errorAssertingFunc = "ERROR GETTING GET_COLOR FUNCTION FROM SCRIPT"
	errorCallingFunc   = "ERROR CALLING GET_COLOR FUNCTION"
	errorInvalidHex    = "INVALID HEX CODE"
)

func createColorCallback(script string) (func(float64) (string, error), error) {
	if script == "" {
		return nil, errors.New(errorEmptyScript)
	}

	vm := goja.New()
	_, err := vm.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errorRunningScript, err)
	}

	get_color, ok := goja.AssertFunction(vm.Get("get_color"))
	if !ok {
		return nil, errors.New(errorAssertingFunc)
	}
	res, err := get_color(goja.Undefined(), vm.ToValue(1.0))
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errorCallingFunc, err)
	}

	hexcode := res.String()
	if len(hexcode) != 7 || hexcode[0] != '#' {
		return nil, errors.New(errorInvalidHex)
	}

	return func(value float64) (string, error) {
		res, err := get_color(goja.Undefined(), vm.ToValue(value))
		if err != nil {
			return "", err
		}
		return res.String(), nil
	}, nil
}
