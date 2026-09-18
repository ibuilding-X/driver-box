package crontab

import (
	"testing"
	"time"
)

func TestParseDurationOrDefault(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fallback  time.Duration
		want      time.Duration
		wantError bool
	}{
		{name: "empty uses fallback", value: "", fallback: time.Second, want: time.Second},
		{name: "custom duration", value: "500ms", fallback: time.Second, want: 500 * time.Millisecond},
		{name: "invalid duration", value: "fast", fallback: time.Second, wantError: true},
		{name: "zero duration", value: "0s", fallback: time.Second, wantError: true},
		{name: "negative duration", value: "-1s", fallback: time.Second, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDurationOrDefault(tt.value, tt.fallback)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
