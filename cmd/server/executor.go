package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dop251/goja"
)

var ErrExecutionTimeout = errors.New("JS execution timed out")

func executeNativeJS(ctx context.Context, code string, input map[string]interface{}) (string, error) {
	vm := goja.New()

	// Add 'log' global for debugging in Native Actions
	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		msg := ""
		for _, arg := range call.Arguments {
			msg += fmt.Sprintf("%v ", arg.Export())
		}
		log.Printf("[SANDBOX LOG]: %s", strings.TrimSpace(msg))
		return goja.Undefined()
	})

	// Clear dangerous globals to restrict scope
	vm.Set("process", goja.Undefined())
	vm.Set("require", goja.Undefined())
	vm.Set("fetch", goja.Undefined())
	vm.Set("setTimeout", goja.Undefined())
	vm.Set("setInterval", goja.Undefined())
	vm.Set("globalThis", goja.Undefined())

	// Set execution limit
	timeout := CurrentSystemSettings.GetJSTimeout()
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
		return "", fmt.Errorf("JS error: %w", err)
	}

	return val.String(), nil
}
