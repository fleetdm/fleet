package fleet

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculateHostVitalsQueryDomesticVitals verifies that "domestic" host
// vitals (columns stored directly on the hosts table, e.g. public_ip) produce
// a WHERE clause that filters on hosts.<column> with no foreign-table JOIN, and
// that the LIKE operator is supported in addition to the default "=".
func TestCalculateHostVitalsQueryDomesticVitals(t *testing.T) {
	testCases := []struct {
		name          string
		vital         string
		operator      *HostVitalOperator
		value         string
		expectedQuery string
	}{
		{
			name:          "public_ip equals",
			vital:         "public_ip",
			value:         "203.0.113.10",
			expectedQuery: "SELECT %s FROM %s WHERE hosts.public_ip = ? GROUP BY hosts.id",
		},
		{
			name:          "public_ip LIKE",
			vital:         "public_ip",
			operator:      new(HostVitalOperatorLike),
			value:         "203.0.113.%",
			expectedQuery: "SELECT %s FROM %s WHERE hosts.public_ip LIKE ? GROUP BY hosts.id",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			criteria := &HostVitalCriteria{
				Vital:    new(tt.vital),
				Operator: tt.operator,
				Value:    new(tt.value),
			}
			criteriaJSON, err := json.Marshal(criteria)
			require.NoError(t, err)

			label := &Label{HostVitalsCriteria: new(json.RawMessage(criteriaJSON))}
			query, values, err := label.CalculateHostVitalsQuery()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedQuery, query)
			assert.Equal(t, []any{tt.value}, values)
		})
	}
}

// TestCalculateHostVitalsQueryRejectsBadOperator ensures unsupported operators
// are still rejected after LIKE is allowed.
func TestCalculateHostVitalsQueryRejectsBadOperator(t *testing.T) {
	criteria := &HostVitalCriteria{
		Vital:    new("public_ip"),
		Operator: new(HostVitalOperatorGreater),
		Value:    new("203.0.113.10"),
	}
	criteriaJSON, err := json.Marshal(criteria)
	require.NoError(t, err)

	label := &Label{HostVitalsCriteria: new(json.RawMessage(criteriaJSON))}
	_, _, err = label.CalculateHostVitalsQuery()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operator")
}

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
