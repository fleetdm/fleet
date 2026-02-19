package endpointer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONKeyRewriteReader_OldKeyPassThrough(t *testing.T) {
	// Old (deprecated) key should pass through as-is and be tracked.
	input := `{"team_id": 42, "name": "hello"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(42), result["team_id"])
	assert.Equal(t, "hello", result["name"])
	assert.Nil(t, result["fleet_id"], "new key should not appear in output")

	// Verify deprecated key was tracked.
	assert.Equal(t, []string{"team_id"}, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_NewKeyRewritten(t *testing.T) {
	// New key should be rewritten to old key for struct deserialization.
	input := `{"fleet_id": 42, "name": "hello"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(42), result["team_id"])
	assert.Nil(t, result["fleet_id"], "new key should be rewritten to old")
	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_NoRewriteNeeded(t *testing.T) {
	// Unrelated keys should pass through unchanged.
	input := `{"other_field": 42, "name": "hello"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(42), result["other_field"])
	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_AliasConflict(t *testing.T) {
	input := `{"team_id": 42, "fleet_id": 99}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	_, err := io.ReadAll(r)
	require.Error(t, err)

	var ace *AliasConflictError
	require.True(t, errors.As(err, &ace))
	assert.Equal(t, "team_id", ace.Old)
	assert.Equal(t, "fleet_id", ace.New)
}

func TestJSONKeyRewriteReader_AliasConflictNewThenOld(t *testing.T) {
	// New key first, then deprecated key.
	input := `{"fleet_id": 99, "team_id": 42}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	_, err := io.ReadAll(r)
	require.Error(t, err)

	var ace *AliasConflictError
	require.True(t, errors.As(err, &ace))
}

func TestJSONKeyRewriteReader_NestedObjects(t *testing.T) {
	input := `{"outer": {"team_id": 1}, "team_id": 2}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	// Old keys should pass through as-is.
	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(2), result["team_id"])
	inner := result["outer"].(map[string]any)
	assert.Equal(t, float64(1), inner["team_id"])

	assert.Contains(t, r.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_NestedNewKeys(t *testing.T) {
	input := `{"outer": {"fleet_id": 1}, "fleet_id": 2}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	// New keys should be rewritten to old keys.
	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(2), result["team_id"])
	inner := result["outer"].(map[string]any)
	assert.Equal(t, float64(1), inner["team_id"])

	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_NestedConflictDoesNotAffectOuter(t *testing.T) {
	// Conflict in nested object should be detected, even though outer is fine.
	input := `{"name": "ok", "inner": {"team_id": 1, "fleet_id": 2}}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	_, err := io.ReadAll(r)
	require.Error(t, err)

	var ace *AliasConflictError
	require.True(t, errors.As(err, &ace))
}

func TestJSONKeyRewriteReader_NoConflictAcrossScopes(t *testing.T) {
	// team_id in outer, fleet_id in inner — no conflict (different scopes).
	input := `{"team_id": 1, "inner": {"fleet_id": 2}}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(1), result["team_id"])
	inner := result["inner"].(map[string]any)
	assert.Equal(t, float64(2), inner["team_id"]) // fleet_id rewritten to team_id
}

func TestJSONKeyRewriteReader_StringValuesNotRewritten(t *testing.T) {
	// "team_id" as a string value (not a key) should NOT be rewritten.
	input := `{"name": "team_id", "description": "the team_id field"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	// String values should not be rewritten, only keys.
	assert.Equal(t, "team_id", result["name"], "string value should not be rewritten")
	assert.Equal(t, "the team_id field", result["description"])
	// Make sure it didn't accidentally transform the team_id string value into a new fleet_id key.
	assert.Empty(t, result["fleet_id"], "new key should not appear in output")
	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_ArrayValues(t *testing.T) {
	input := `{"team_id": [1, 2, 3], "items": ["a", "b"]}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.NotNil(t, result["team_id"])
	assert.Contains(t, r.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_ArrayOfObjects(t *testing.T) {
	input := `{"items": [{"team_id": 1}, {"team_id": 2}]}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	items := result["items"].([]any)
	for _, item := range items {
		obj := item.(map[string]any)
		assert.NotNil(t, obj["team_id"])
		assert.Nil(t, obj["fleet_id"])
	}
}

func TestJSONKeyRewriteReader_MultipleRules(t *testing.T) {
	input := `{"team_id": 1, "team_name": "Engineering"}`
	rules := []AliasRule{
		{OldKey: "team_id", NewKey: "fleet_id"},
		{OldKey: "team_name", NewKey: "fleet_name"},
	}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(1), result["team_id"])
	assert.Equal(t, "Engineering", result["team_name"])
	assert.Nil(t, result["fleet_id"])
	assert.Nil(t, result["fleet_name"])

	deprecated := r.UsedDeprecatedKeys()
	assert.Len(t, deprecated, 2)
	assert.Contains(t, deprecated, "team_id")
	assert.Contains(t, deprecated, "team_name")
}

func TestJSONKeyRewriteReader_MultipleRulesNewKeys(t *testing.T) {
	// New keys should be rewritten to old keys.
	input := `{"fleet_id": 1, "fleet_name": "Engineering"}`
	rules := []AliasRule{
		{OldKey: "team_id", NewKey: "fleet_id"},
		{OldKey: "team_name", NewKey: "fleet_name"},
	}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(1), result["team_id"])
	assert.Equal(t, "Engineering", result["team_name"])
	assert.Nil(t, result["fleet_id"])
	assert.Nil(t, result["fleet_name"])

	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_EmptyObject(t *testing.T) {
	input := `{}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(out))
	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_NullValues(t *testing.T) {
	input := `{"team_id": null}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Contains(t, result, "team_id")
	assert.Nil(t, result["team_id"])
}

func TestJSONKeyRewriteReader_BooleanValues(t *testing.T) {
	input := `{"team_id": true}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, true, result["team_id"])
}

func TestJSONKeyRewriteReader_NoRules(t *testing.T) {
	input := `{"team_id": 42}`
	var rules []AliasRule

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	// With no rules, output should be identical to input.
	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(42), result["team_id"])
	assert.Empty(t, r.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_LargePayload(t *testing.T) {
	// Build a large JSON payload that exceeds the internal buffer size (4096 bytes).
	var sb strings.Builder
	sb.WriteString(`{"team_id": 1`)
	for i := range 500 {
		sb.WriteString(fmt.Sprintf(`, "field_%04d": "value"`, i))
	}
	sb.WriteString(`}`)
	input := sb.String()

	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, float64(1), result["team_id"])
	assert.Nil(t, result["fleet_id"])
	assert.Contains(t, r.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_WithJSONDecoderOldKey(t *testing.T) {
	// Simulate the real usage: json.NewDecoder reading from the rewriter
	// with old (deprecated) key in the request. The struct uses old key names.
	input := `{"team_id": 42, "name": "test"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	rewriter := NewJSONKeyRewriteReader(strings.NewReader(input), rules)

	type request struct {
		TeamID int    `json:"team_id"`
		Name   string `json:"name"`
	}
	var req request
	err := json.NewDecoder(rewriter).Decode(&req)
	require.NoError(t, err)
	assert.Equal(t, 42, req.TeamID)
	assert.Equal(t, "test", req.Name)
	assert.Contains(t, rewriter.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_WithJSONDecoderNewKey(t *testing.T) {
	// Simulate the real usage: json.NewDecoder reading from the rewriter
	// with new key in the request. Should be rewritten to old key.
	input := `{"fleet_id": 42, "name": "test"}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	rewriter := NewJSONKeyRewriteReader(strings.NewReader(input), rules)

	type request struct {
		TeamID int    `json:"team_id"`
		Name   string `json:"name"`
	}
	var req request
	err := json.NewDecoder(rewriter).Decode(&req)
	require.NoError(t, err)
	assert.Equal(t, 42, req.TeamID)
	assert.Equal(t, "test", req.Name)
	assert.Empty(t, rewriter.UsedDeprecatedKeys())
}

func TestJSONKeyRewriteReader_AliasConflictWithJSONDecoder(t *testing.T) {
	input := `{"team_id": 42, "fleet_id": 99}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	rewriter := NewJSONKeyRewriteReader(strings.NewReader(input), rules)

	type request struct {
		TeamID int `json:"team_id"`
	}
	var req request
	err := json.NewDecoder(rewriter).Decode(&req)
	require.Error(t, err)

	var ace *AliasConflictError
	require.True(t, errors.As(err, &ace))
	assert.Equal(t, "team_id", ace.Old)
	assert.Equal(t, "fleet_id", ace.New)
}

func TestJSONKeyRewriteReader_DeeplyNestedObjectsOldKeys(t *testing.T) {
	input := `{"a": {"b": {"c": {"team_id": 99}}}}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	inner := result["a"].(map[string]any)["b"].(map[string]any)["c"].(map[string]any)
	assert.Equal(t, float64(99), inner["team_id"])
	assert.Contains(t, r.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_DeeplyNestedObjectsNewKeys(t *testing.T) {
	input := `{"a": {"b": {"c": {"fleet_id": 99}}}}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	inner := result["a"].(map[string]any)["b"].(map[string]any)["c"].(map[string]any)
	assert.Equal(t, float64(99), inner["team_id"])
}

func TestJSONKeyRewriteReader_TopLevelArrayOldKeys(t *testing.T) {
	input := `[{"team_id": 1}, {"team_id": 2}]`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Len(t, result, 2)
	assert.Equal(t, float64(1), result[0]["team_id"])
	assert.Equal(t, float64(2), result[1]["team_id"])
	assert.Contains(t, r.UsedDeprecatedKeys(), "team_id")
}

func TestJSONKeyRewriteReader_TopLevelArrayNewKeys(t *testing.T) {
	input := `[{"fleet_id": 1}, {"fleet_id": 2}]`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Len(t, result, 2)
	assert.Equal(t, float64(1), result[0]["team_id"])
	assert.Equal(t, float64(2), result[1]["team_id"])
}

func TestJSONKeyRewriteReader_NestedObjectStringValue(t *testing.T) {
	// Object as value with keys that need tracking.
	input := `{"config": {"team_id": 5, "enabled": true}}`
	rules := []AliasRule{{OldKey: "team_id", NewKey: "fleet_id"}}

	r := NewJSONKeyRewriteReader(strings.NewReader(input), rules)
	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	config := result["config"].(map[string]any)
	assert.Equal(t, float64(5), config["team_id"])
	assert.Equal(t, true, config["enabled"])
	assert.Equal(t, []string{"team_id"}, r.UsedDeprecatedKeys())
}

func TestAliasConflictError_ErrorMessage(t *testing.T) {
	err := &AliasConflictError{Old: "team_id", New: "fleet_id"}
	assert.Contains(t, err.Error(), "team_id")
	assert.Contains(t, err.Error(), "fleet_id")
}
