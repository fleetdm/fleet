package fleet

import (
	"reflect"
	"strings"
	"testing"
)

func TestDetectMissingLabels(t *testing.T) {
	labelMap := map[string]uint{"label1": 1, "label2": 2}

	testCases := []struct {
		name     string
		labels   []string
		expected []string
	}{
		{
			name:     "no labels",
			labels:   []string{},
			expected: []string{},
		},
		{
			name:     "empty label",
			labels:   []string{""},
			expected: []string{},
		},
		{
			name:     "one missing label",
			labels:   []string{"iamnotalabel"},
			expected: []string{"iamnotalabel"},
		},
		{
			name:     "missing multiple labels",
			labels:   []string{"mac", "windows"},
			expected: []string{"mac", "windows"},
		},
		{
			name:     "some missing labels and some valid labels",
			labels:   []string{"label1", "mac", "label2", "windows"},
			expected: []string{"mac", "windows"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			missing := DetectMissingLabels(labelMap, tt.labels)
			if !reflect.DeepEqual(tt.expected, missing) {
				t.Errorf("Expected [%s], but got [%s]", strings.Join(tt.expected, ", "), strings.Join(missing, ", "))
			}
		})
	}
}
