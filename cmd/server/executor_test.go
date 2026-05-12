package main

import (
	"context"
	"testing"
)

func TestExecuteNativeJS(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name    string
		code    string
		input   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "simple addition",
			code:  "1 + 2",
			input: nil,
			want:  "3",
		},
		{
			name:  "use input",
			code:  "input.a + input.b",
			input: map[string]interface{}{"a": 10, "b": 20},
			want:  "30",
		},
		{
			name:    "invalid code",
			code:    "invalid code",
			input:   nil,
			want:    "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeNativeJS(ctx, tt.code, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeNativeJS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("executeNativeJS() got = %v, want %v", got, tt.want)
			}
		})
	}
}
