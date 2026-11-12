package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			if !assert.Equal(t, tt.expected, missing) {
				t.Errorf("Expected [%s], but got [%s]", strings.Join(tt.expected, ", "), strings.Join(missing, ", "))
			}
		})
	}
}

func TestLabelOverlap(t *testing.T) {
	testCases := []struct {
		name    string
		include []string
		exclude []string
		want    string
	}{
		{name: "no overlap", include: []string{"a", "b"}, exclude: []string{"c", "d"}, want: ""},
		{name: "both empty", include: nil, exclude: nil, want: ""},
		{name: "empty include", include: nil, exclude: []string{"a"}, want: ""},
		{name: "empty exclude", include: []string{"a"}, exclude: nil, want: ""},
		{name: "single overlap", include: []string{"a", "b"}, exclude: []string{"b", "c"}, want: "b"},
		{name: "returns first overlap by exclude order", include: []string{"a", "b", "c"}, exclude: []string{"x", "c", "a"}, want: "c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, LabelOverlap(tc.include, tc.exclude))
		})
	}
}
