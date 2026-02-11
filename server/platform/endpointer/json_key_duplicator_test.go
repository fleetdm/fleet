package endpointer

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicateJSONKeys(t *testing.T) {
	rules := []AliasRule{
		{OldKey: "team_id", NewKey: "fleet_id"},
		{OldKey: "team_ids", NewKey: "fleet_ids"},
		{OldKey: "team_name", NewKey: "fleet_name"},
	}

	tests := []struct {
		name     string
		input    string
		rules    []AliasRule
		validate func(t *testing.T, result []byte)
	}{
		{
			name:  "BasicDuplication",
			input: `{"fleet_id": 42, "name": "hello"}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(42), m["fleet_id"])
				assert.Equal(t, float64(42), m["team_id"])
				assert.Equal(t, "hello", m["name"])
			},
		},
		{
			name:  "NoDuplicationNeeded",
			input: `{"name": "hello", "count": 5}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				assert.JSONEq(t, `{"name": "hello", "count": 5}`, string(result))
			},
		},
		{
			name:  "MultipleRules",
			input: `{"fleet_id": 1, "fleet_name": "test"}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(1), m["fleet_id"])
				assert.Equal(t, float64(1), m["team_id"])
				assert.Equal(t, "test", m["fleet_name"])
				assert.Equal(t, "test", m["team_name"])
			},
		},
		{
			name:  "OldKeyAlreadyExists",
			input: `{"fleet_id": 1, "team_id": 2}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				// When old key already exists, no duplication should happen.
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(1), m["fleet_id"])
				assert.Equal(t, float64(2), m["team_id"])
			},
		},
		{
			name:  "StringValue",
			input: `{"fleet_name": "my fleet"}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, "my fleet", m["fleet_name"])
				assert.Equal(t, "my fleet", m["team_name"])
			},
		},
		{
			name:  "NullValue",
			input: `{"fleet_id": null}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Nil(t, m["fleet_id"])
				assert.Nil(t, m["team_id"])
				_, hasTeamID := m["team_id"]
				assert.True(t, hasTeamID)
			},
		},
		{
			name:  "BooleanValue",
			input: `{"fleet_id": true}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, true, m["fleet_id"])
				assert.Equal(t, true, m["team_id"])
			},
		},
		{
			name:  "ArrayValue",
			input: `{"fleet_ids": [1, 2, 3]}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, []any{float64(1), float64(2), float64(3)}, m["fleet_ids"])
				assert.Equal(t, []any{float64(1), float64(2), float64(3)}, m["team_ids"])
			},
		},
		{
			name:  "ObjectValue",
			input: `{"fleet_id": {"sub": "value"}}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				expected := map[string]any{"sub": "value"}
				assert.Equal(t, expected, m["fleet_id"])
				assert.Equal(t, expected, m["team_id"])
			},
		},
		{
			name:  "NestedObjects",
			input: `{"outer": {"fleet_id": 10}}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				inner := m["outer"].(map[string]any)
				assert.Equal(t, float64(10), inner["fleet_id"])
				assert.Equal(t, float64(10), inner["team_id"])
			},
		},
		{
			name:  "DeeplyNested",
			input: `{"a": {"b": {"c": {"fleet_id": 99}}}}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				c := m["a"].(map[string]any)["b"].(map[string]any)["c"].(map[string]any)
				assert.Equal(t, float64(99), c["fleet_id"])
				assert.Equal(t, float64(99), c["team_id"])
			},
		},
		{
			// This simulates the ABM tokens response pattern where a duplicated
			// outer key (e.g., ios_fleet→ios_team) has an object value that itself
			// contains keys needing duplication (e.g., fleet_id→team_id).
			name:  "DuplicatedKeyWithNestedDuplicatableKeys",
			input: `{"ios_fleet": {"name": "Default", "fleet_id": 5}}`,
			rules: []AliasRule{
				{OldKey: "team_id", NewKey: "fleet_id"},
				{OldKey: "ios_team", NewKey: "ios_fleet"},
			},
			validate: func(t *testing.T, result []byte) {
				assert.True(t, json.Valid(result), "result should be valid JSON: %s", string(result))
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				// Both ios_fleet and ios_team should exist.
				iosFleet := m["ios_fleet"].(map[string]any)
				iosTeam := m["ios_team"].(map[string]any)
				// Both should have fleet_id AND team_id.
				assert.Equal(t, "Default", iosFleet["name"])
				assert.Equal(t, float64(5), iosFleet["fleet_id"])
				assert.Equal(t, float64(5), iosFleet["team_id"])
				assert.Equal(t, "Default", iosTeam["name"])
				assert.Equal(t, float64(5), iosTeam["fleet_id"])
				assert.Equal(t, float64(5), iosTeam["team_id"])
			},
		},
		{
			name:  "ArrayOfObjects",
			input: `[{"fleet_id": 1}, {"fleet_id": 2}]`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var arr []map[string]any
				require.NoError(t, json.Unmarshal(result, &arr))
				require.Len(t, arr, 2)
				assert.Equal(t, float64(1), arr[0]["fleet_id"])
				assert.Equal(t, float64(1), arr[0]["team_id"])
				assert.Equal(t, float64(2), arr[1]["fleet_id"])
				assert.Equal(t, float64(2), arr[1]["team_id"])
			},
		},
		{
			name:  "ScopeIsolation",
			input: `{"fleet_id": 1, "child": {"team_id": 5}}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				// Top level: fleet_id should be duplicated (no team_id at top level).
				assert.Equal(t, float64(1), m["fleet_id"])
				assert.Equal(t, float64(1), m["team_id"])
				// Child: team_id exists but fleet_id doesn't; no duplication
				// (we only duplicate new->old, not old->new).
				child := m["child"].(map[string]any)
				assert.Equal(t, float64(5), child["team_id"])
				_, hasFleetIDInChild := child["fleet_id"]
				assert.False(t, hasFleetIDInChild)
			},
		},
		{
			name:  "EmptyObject",
			input: `{}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				assert.JSONEq(t, `{}`, string(result))
			},
		},
		{
			name:  "EmptyArray",
			input: `[]`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				assert.JSONEq(t, `[]`, string(result))
			},
		},
		{
			name:  "NoRules",
			input: `{"fleet_id": 42}`,
			rules: nil,
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, `{"fleet_id": 42}`, string(result))
			},
		},
		{
			name:  "EmptyData",
			input: ``,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				assert.Equal(t, ``, string(result))
			},
		},
		{
			name:  "StringValueNotDuplicated",
			input: `{"value": "fleet_id"}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				// String values that happen to match a key name should NOT trigger duplication.
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, "fleet_id", m["value"])
				_, hasTeamID := m["team_id"]
				assert.False(t, hasTeamID)
			},
		},
		{
			name:  "NumberWithExponent",
			input: `{"fleet_id": 1.5e2}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(150), m["fleet_id"])
				assert.Equal(t, float64(150), m["team_id"])
			},
		},
		{
			name:  "NegativeNumber",
			input: `{"fleet_id": -7}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(-7), m["fleet_id"])
				assert.Equal(t, float64(-7), m["team_id"])
			},
		},
		{
			name:  "EscapedQuotesInStringValue",
			input: `{"fleet_name": "he said \"hi\""}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, `he said "hi"`, m["fleet_name"])
				assert.Equal(t, `he said "hi"`, m["team_name"])
			},
		},
		{
			name:  "PrettyPrintedJSON",
			input: "{\n  \"fleet_id\": 42,\n  \"name\": \"test\"\n}",
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(42), m["fleet_id"])
				assert.Equal(t, float64(42), m["team_id"])
				assert.Equal(t, "test", m["name"])
			},
		},
		{
			name:  "ValidJSON",
			input: `{"fleet_id": 42, "nested": {"fleet_name": "x"}, "arr": [{"fleet_ids": [1]}]}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				// Ensure the result is valid JSON.
				assert.True(t, json.Valid(result), "result should be valid JSON: %s", string(result))

				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(42), m["fleet_id"])
				assert.Equal(t, float64(42), m["team_id"])

				nested := m["nested"].(map[string]any)
				assert.Equal(t, "x", nested["fleet_name"])
				assert.Equal(t, "x", nested["team_name"])

				arr := m["arr"].([]any)
				arrObj := arr[0].(map[string]any)
				assert.Equal(t, []any{float64(1)}, arrObj["fleet_ids"])
				assert.Equal(t, []any{float64(1)}, arrObj["team_ids"])
			},
		},
		{
			name: "LargePayload",
			input: func() string {
				var items []string
				for i := range 100 {
					items = append(items, fmt.Sprintf(`{"fleet_id": %d, "field_%04d": "val"}`, i, i))
				}
				return "[" + strings.Join(items, ",") + "]"
			}(),
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				assert.True(t, json.Valid(result), "result should be valid JSON")
				var arr []map[string]any
				require.NoError(t, json.Unmarshal(result, &arr))
				require.Len(t, arr, 100)
				for i, obj := range arr {
					assert.Equal(t, float64(i), obj["fleet_id"])
					assert.Equal(t, float64(i), obj["team_id"])
				}
			},
		},
		{
			name:  "OnlyOldKeyPresent",
			input: `{"team_id": 5}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				// The old key is present but not the new key. We only duplicate
				// new->old, not old->new. So no duplication should happen.
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(5), m["team_id"])
				_, hasFleetID := m["fleet_id"]
				assert.False(t, hasFleetID)
			},
		},
		{
			name:  "MixedKeysAcrossScopes",
			input: `{"fleet_id": 1, "child": {"fleet_id": 2, "team_id": 3}}`,
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				// Top level: fleet_id duplicated (no team_id at top).
				assert.Equal(t, float64(1), m["fleet_id"])
				assert.Equal(t, float64(1), m["team_id"])
				// Child: both keys exist, no duplication.
				child := m["child"].(map[string]any)
				assert.Equal(t, float64(2), child["fleet_id"])
				assert.Equal(t, float64(3), child["team_id"])
			},
		},
		{
			name:  "TrailingNewline",
			input: "{\"fleet_id\": 1}\n",
			rules: rules,
			validate: func(t *testing.T, result []byte) {
				// json.Encoder appends a newline; ensure it still works.
				var m map[string]any
				require.NoError(t, json.Unmarshal(result, &m))
				assert.Equal(t, float64(1), m["fleet_id"])
				assert.Equal(t, float64(1), m["team_id"])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DuplicateJSONKeys([]byte(tc.input), tc.rules)
			tc.validate(t, result)
		})
	}
}

// TestDuplicateJSONKeysWithEncoder tests that the duplicator works correctly
// with the output of json.Encoder (which adds pretty-printing and a trailing newline).
func TestDuplicateJSONKeysWithEncoder(t *testing.T) {
	rules := []AliasRule{
		{OldKey: "team_id", NewKey: "fleet_id"},
	}

	type response struct {
		FleetID int    `json:"fleet_id"`
		Name    string `json:"name"`
	}

	data, err := json.MarshalIndent(response{FleetID: 42, Name: "test"}, "", "  ")
	require.NoError(t, err)

	result := DuplicateJSONKeys(data, rules)
	assert.True(t, json.Valid(result), "result should be valid JSON: %s", string(result))

	var m map[string]any
	require.NoError(t, json.Unmarshal(result, &m))
	assert.Equal(t, float64(42), m["fleet_id"])
	assert.Equal(t, float64(42), m["team_id"])
	assert.Equal(t, "test", m["name"])
}

// TestDuplicateJSONKeysIdempotent ensures that running the duplicator twice
// doesn't add more keys (since after the first run the old key exists).
func TestDuplicateJSONKeysIdempotent(t *testing.T) {
	rules := []AliasRule{
		{OldKey: "team_id", NewKey: "fleet_id"},
	}

	input := `{"fleet_id": 42}`
	first := DuplicateJSONKeys([]byte(input), rules)

	var m1 map[string]any
	require.NoError(t, json.Unmarshal(first, &m1))
	assert.Equal(t, float64(42), m1["fleet_id"])
	assert.Equal(t, float64(42), m1["team_id"])

	// Second pass should not add anything new.
	second := DuplicateJSONKeys(first, rules)
	var m2 map[string]any
	require.NoError(t, json.Unmarshal(second, &m2))
	assert.Equal(t, float64(42), m2["fleet_id"])
	assert.Equal(t, float64(42), m2["team_id"])
	assert.Len(t, m2, 2)
}
