package log

import (
	"testing"
)

func TestGetLevel(t *testing.T) {
	for _, levelStr := range level {
		num := GetLevel(levelStr)
		if num < 0 || num > 7 {
			t.Fatalf("get log level failed: %d\n", num)
		}
	}

	num := GetLevel("default")
	if num != 7 {
		t.Errorf("defaultLogLevel set failed: %d\n", num)
	}
}
