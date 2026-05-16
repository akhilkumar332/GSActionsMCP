package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

var ErrExecutionTimeout = errors.New("JS execution timed out")

func executeNativeJS(ctx context.Context, code string, input map[string]interface{}) (string, error) {
	vm := goja.New()

	// Set execution limit
	timeout := 5 * time.Second
	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt(ErrExecutionTimeout)
	})
	defer timer.Stop()

	// Set the input object in the JS environment
	if err := vm.Set("input", input); err != nil {
		return "", fmt.Errorf("failed to set input: %w", err)
	}

	// Run the provided JS code
	val, err := vm.RunString(code)
	if err != nil {
		// Check if it's our timeout error
		var jsErr *goja.InterruptedError
		if errors.As(err, &jsErr) && jsErr.Value() == ErrExecutionTimeout {
			return "", ErrExecutionTimeout
		}
		return "", err
	}

	return val.String(), nil
}
