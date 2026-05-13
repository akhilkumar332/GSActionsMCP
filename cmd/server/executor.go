package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/robertkrimen/otto"
)

var ErrExecutionTimeout = errors.New("JS execution timed out")

func executeNativeJS(ctx context.Context, code string, input map[string]interface{}) (string, error) {
	vm := otto.New()

	// Set execution limit
	vm.Interrupt = make(chan func(), 1)
	timeout := 5 * time.Second
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt <- func() {
			panic(ErrExecutionTimeout)
		}
	})
	defer timer.Stop()

	// Set the input object in the JS environment
	if err := vm.Set("input", input); err != nil {
		return "", fmt.Errorf("failed to set input: %w", err)
	}

	// Run the provided JS code with panic recovery for timeout
	var val otto.Value
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				if r == ErrExecutionTimeout {
					err = ErrExecutionTimeout
				} else {
					panic(r) // Re-panic if it's not our timeout
				}
			}
		}()
		val, err = vm.Run(code)
	}()

	if err != nil {
		return "", err
	}

	return val.String(), nil
}
