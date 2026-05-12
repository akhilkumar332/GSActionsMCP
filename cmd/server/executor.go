package main

import (
	"context"
	"github.com/robertkrimen/otto"
)

func executeNativeJS(ctx context.Context, code string, input map[string]interface{}) (string, error) {
	vm := otto.New()
	
	// Set the input object in the JS environment
	if err := vm.Set("input", input); err != nil {
		return "", err
	}
	
	// Run the provided JS code
	val, err := vm.Run(code)
	if err != nil {
		return "", err
	}
	
	return val.String(), nil
}
