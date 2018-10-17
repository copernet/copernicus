package util

import (
	"reflect"
	"testing"
)

func TestNewFastRandomContext(t *testing.T) {
	tests := []struct {
		in   bool
		want *FastRandomContext
	}{
		{true, &FastRandomContext{11, 11}},
	}

	for i, v := range tests {
		value := v
		result := NewFastRandomContext(value.in)
		if !reflect.DeepEqual(result, value.want) {
			t.Errorf("The %d not expect.", i)
		}
		result.Rand32()
	}

	f := NewFastRandomContext(false)
	f.Rand32()
}
