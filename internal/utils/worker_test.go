package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkerCount(t *testing.T) {
	cases := []struct {
		name  string
		extra map[string]string
		def   int
		max   int
		want  int
	}{
		{"no extra returns default", nil, 4, 16, 4},
		{"missing key returns default", map[string]string{"foo": "bar"}, 4, 16, 4},
		{"valid value", map[string]string{"workers": "8"}, 4, 16, 8},
		{"non-numeric falls back", map[string]string{"workers": "abc"}, 4, 16, 4},
		{"zero falls back", map[string]string{"workers": "0"}, 4, 16, 4},
		{"negative falls back", map[string]string{"workers": "-1"}, 4, 16, 4},
		{"above max falls back", map[string]string{"workers": "32"}, 4, 16, 4},
		{"equal to max ok", map[string]string{"workers": "16"}, 4, 16, 16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, WorkerCount(c.extra, c.def, c.max))
		})
	}
}
