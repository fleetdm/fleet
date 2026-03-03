package spec

import (
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownJSONKeys(t *testing.T) {
	t.Parallel()

	t.Run("simple struct", func(t *testing.T) {
		type Simple struct {
			Name string `json:"name"`
			Age  int    `json:"age,omitempty"`
		}
		keys := knownJSONKeys(reflect.TypeOf(Simple{}))
		require.Len(t, keys, 2)
		assert.Contains(t, keys, "name")
		assert.Contains(t, keys, "age")
	})

	t.Run("embedded struct", func(t *testing.T) {
		// Label embeds BaseItem and fleet.LabelSpec
		keys := knownJSONKeys(reflect.TypeOf(Label{}))
		// Should have path/paths from BaseItem + all LabelSpec fields
		assert.Contains(t, keys, "path")
		assert.Contains(t, keys, "paths")
		assert.Contains(t, keys, "name")
		assert.Contains(t, keys, "query")
		assert.Contains(t, keys, "description")
	})

	t.Run("json dash excluded", func(t *testing.T) {
		type WithDash struct {
			Name    string `json:"name"`
			Ignored string `json:"-"`
		}
		keys := knownJSONKeys(reflect.TypeOf(WithDash{}))
		assert.Contains(t, keys, "name")
		assert.NotContains(t, keys, "-")
		assert.NotContains(t, keys, "Ignored")
	})

	t.Run("no json tag excluded", func(t *testing.T) {
		type NoTag struct {
			Name    string `json:"name"`
			NoTag   string
			Defined bool
		}
		keys := knownJSONKeys(reflect.TypeOf(NoTag{}))
		assert.Contains(t, keys, "name")
		assert.NotContains(t, keys, "NoTag")
		assert.NotContains(t, keys, "Defined")
	})

	t.Run("pointer type", func(t *testing.T) {
		keys := knownJSONKeys(reflect.TypeOf(&GitOpsControls{}))
		assert.Contains(t, keys, "macos_updates")
		assert.Contains(t, keys, "scripts")
		// From BaseItem
		assert.Contains(t, keys, "path")
	})

	t.Run("caching works", func(t *testing.T) {
		t1 := reflect.TypeOf(Label{})
		keys1 := knownJSONKeys(t1)
		keys2 := knownJSONKeys(t1)
		// Should return the same map (pointer equality after caching)
		assert.Equal(t, keys1, keys2)
	})
}

func TestValidateUnknownKeys(t *testing.T) {
	t.Parallel()

	t.Run("no unknown keys", func(t *testing.T) {
		data := map[string]interface{}{
			"name":        "test-query",
			"query":       "SELECT 1",
			"description": "a test",
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(fleet.QuerySpec{}), []string{"reports", "[0]"}, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("unknown key detected", func(t *testing.T) {
		data := map[string]interface{}{
			"name":                "test-query",
			"query":               "SELECT 1",
			"unknown_field":       "bad",
			"other_unknown_field": "also bad",
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(fleet.QuerySpec{}), []string{"reports", "[0]"}, "test.yml")
		require.Len(t, errs, 2)
		assert.Contains(t, errs[0].Error(), "unknown_field")
		assert.Contains(t, errs[0].Error(), "reports.[0]")
		assert.Contains(t, errs[0].Error(), "test.yml")
		assert.Contains(t, errs[1].Error(), "other_unknown_field")
		assert.Contains(t, errs[1].Error(), "reports.[0]")
		assert.Contains(t, errs[1].Error(), "test.yml")

		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Equal(t, "unknown_field", unknownErr.Field)
	})

	t.Run("multiple unknown keys", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"bad1": "x",
			"bad2": "y",
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(fleet.QuerySpec{}), []string{"reports"}, "test.yml")
		assert.Len(t, errs, 2)
	})

	t.Run("scalar data no errors", func(t *testing.T) {
		errs := validateUnknownKeys("just a string", reflect.TypeOf(fleet.QuerySpec{}), nil, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("nil data no errors", func(t *testing.T) {
		errs := validateUnknownKeys(nil, reflect.TypeOf(fleet.QuerySpec{}), nil, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("nested struct validation", func(t *testing.T) {
		// MacOSSetup is a pointer-to-struct field in GitOpsControls
		data := map[string]interface{}{
			"macos_setup": map[string]interface{}{
				"bootstrap_package": "pkg.pkg",
				"bad_nested_field":  true,
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(GitOpsControls{}), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "bad_nested_field")
		assert.Contains(t, errs[0].Error(), "controls.macos_setup")
	})

	t.Run("slice validation", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"name":      "query1",
				"query":     "SELECT 1",
				"bad_field": "x",
			},
			map[string]interface{}{
				"name":  "query2",
				"query": "SELECT 2",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf([]Query{}), []string{"reports"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "bad_field")
	})

	t.Run("non-struct target type", func(t *testing.T) {
		data := map[string]interface{}{
			"anything": "goes",
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(""), nil, "test.yml")
		assert.Empty(t, errs)
	})
}

func TestAnyFieldTypeRegistry(t *testing.T) {
	t.Parallel()

	t.Run("controls any-field recursion", func(t *testing.T) {
		data := map[string]interface{}{
			"macos_updates": map[string]interface{}{
				"minimum_version": "14.0",
				"deadline":        "2024-01-01",
				"deadlinee":       "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(GitOpsControls{}), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "deadlinee")
		assert.Contains(t, errs[0].Error(), "controls.macos_updates")
	})

	t.Run("windows_updates any-field recursion", func(t *testing.T) {
		data := map[string]interface{}{
			"windows_updates": map[string]interface{}{
				"deadline_days":    5,
				"grace_period_das": 2, // typo
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(GitOpsControls{}), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "grace_period_das")
	})

	t.Run("macos_settings any-field recursion", func(t *testing.T) {
		data := map[string]interface{}{
			"macos_settings": map[string]interface{}{
				"custom_settings":  []interface{}{},
				"custom_settingss": "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(GitOpsControls{}), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "custom_settingss")
	})

	t.Run("android_settings any-field recursion", func(t *testing.T) {
		data := map[string]interface{}{
			"android_settings": map[string]interface{}{
				"custom_settings": []interface{}{},
				"certificatess":   "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeOf(GitOpsControls{}), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "certificatess")
	})

	t.Run("all registered types present", func(t *testing.T) {
		overrides, ok := anyFieldTypes[reflect.TypeOf(GitOpsControls{})]
		require.True(t, ok)
		assert.Contains(t, overrides, "macos_updates")
		assert.Contains(t, overrides, "ios_updates")
		assert.Contains(t, overrides, "ipados_updates")
		assert.Contains(t, overrides, "macos_migration")
		assert.Contains(t, overrides, "windows_updates")
		assert.Contains(t, overrides, "macos_settings")
		assert.Contains(t, overrides, "windows_settings")
		assert.Contains(t, overrides, "android_settings")
	})
}

func TestValidateRawKeys(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		raw := []byte(`{"name":"test","query":"SELECT 1"}`)
		errs := validateRawKeys(raw, reflect.TypeOf(fleet.QuerySpec{}), "test.yml", []string{"reports"})
		assert.Empty(t, errs)
	})

	t.Run("unknown key in json", func(t *testing.T) {
		raw := []byte(`{"name":"test","query":"SELECT 1","typo_field":"bad"}`)
		errs := validateRawKeys(raw, reflect.TypeOf(fleet.QuerySpec{}), "test.yml", []string{"reports"})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "typo_field")
	})

	t.Run("invalid json no error", func(t *testing.T) {
		raw := []byte(`{invalid json`)
		errs := validateRawKeys(raw, reflect.TypeOf(fleet.QuerySpec{}), "test.yml", []string{"reports"})
		assert.Empty(t, errs) // parse errors handled elsewhere
	})
}
