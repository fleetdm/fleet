package fleet

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAndroidAgentVitalsFieldDefinitions guards the rules the reflection
// helpers and host_android_agent_vitals rely on, so a new vital that breaks
// one fails here rather than at runtime.
func TestAndroidAgentVitalsFieldDefinitions(t *testing.T) {
	typ := reflect.TypeFor[AndroidAgentVitals]()
	require.Positive(t, typ.NumField())
	for field := range typ.Fields() {
		t.Run(field.Name, func(t *testing.T) {
			require.Equal(t, reflect.Pointer, field.Type.Kind(), "must be a pointer so a missing vital is distinguishable")
			assert.Contains(t, []reflect.Kind{reflect.Bool, reflect.Int64, reflect.String}, field.Type.Elem().Kind())

			column := field.Tag.Get("db")
			require.NotEmpty(t, column)
			assert.Regexp(t, `^[a-z][a-z0-9_]*$`, column, "used unquoted in SQL")
			assert.Equal(t, column+",omitempty", field.Tag.Get("json"), "the API name must match the column")
			assert.Equal(t, "-", field.Tag.Get("csv"), "gocsv flattens embedded host fields into the hosts CSV export")
			assert.Contains(t, []string{"", "hide"}, field.Tag.Get("personal"))
		})
	}
}

type testAgentVitals struct {
	Flag    *bool   `db:"flag" personal:"hide"`
	Count   *int64  `db:"count"`
	Name    *string `db:"name"`
	Private *string `db:"private" personal:"hide"`
}

func TestAndroidAgentVitalColumns(t *testing.T) {
	longName := strings.Repeat("é", MDMAndroidDeviceVitalMaxLength+10)

	cases := []struct {
		name        string
		vitals      testAgentVitals
		wantColumns []string
		wantValues  []any
	}{
		{"none", testAgentVitals{}, nil, nil},
		{"false and zero are reported", testAgentVitals{Flag: new(false), Count: new(int64(0))}, []string{"flag", "count"}, []any{false, int64(0)}},
		{"only present fields", testAgentVitals{Name: new("pixel")}, []string{"name"}, []any{"pixel"}},
		{
			"strings truncated by character",
			testAgentVitals{Name: &longName},
			[]string{"name"},
			[]any{strings.Repeat("é", MDMAndroidDeviceVitalMaxLength)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			columns, values := vitalColumns(c.vitals)
			require.Equal(t, c.wantColumns, columns)
			require.Equal(t, c.wantValues, values)
		})
	}
}

func TestClearPersonalVitals(t *testing.T) {
	vitals := testAgentVitals{Flag: new(true), Count: new(int64(3)), Name: new("pixel"), Private: new("secret")}
	clearPersonalVitals(&vitals)
	require.Equal(t, testAgentVitals{Count: new(int64(3)), Name: new("pixel")}, vitals)

	roaming := AndroidAgentVitals{Roaming: new(true)}
	roaming.ClearPersonal()
	require.Nil(t, roaming.Roaming, "roaming reveals the owner's travel")
}

func TestAndroidAgentVitalsJSON(t *testing.T) {
	var vitals AndroidAgentVitals
	require.NoError(t, json.Unmarshal([]byte(`{"roaming": false, "vital_from_a_newer_agent": "x"}`), &vitals))
	require.NotNil(t, vitals.Roaming)
	require.False(t, *vitals.Roaming)
	require.False(t, vitals.IsEmpty())

	vitals = AndroidAgentVitals{}
	require.NoError(t, json.Unmarshal([]byte(`{"roaming": null}`), &vitals))
	require.True(t, vitals.IsEmpty(), "null is treated as not reported")
	require.Equal(t, []string{"roaming"}, AndroidAgentVitalColumns())
}
