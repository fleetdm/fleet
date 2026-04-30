package spec

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/hashicorp/go-multierror"
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
		keys := knownJSONKeys(reflect.TypeFor[Simple]())
		require.Len(t, keys, 2)
		assert.Contains(t, keys, "name")
		assert.Contains(t, keys, "age")
	})

	t.Run("embedded struct", func(t *testing.T) {
		// Label embeds BaseItem and fleet.LabelSpec
		keys := knownJSONKeys(reflect.TypeFor[Label]())
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
		keys := knownJSONKeys(reflect.TypeFor[WithDash]())
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
		keys := knownJSONKeys(reflect.TypeFor[NoTag]())
		assert.Contains(t, keys, "name")
		assert.NotContains(t, keys, "NoTag")
		assert.NotContains(t, keys, "Defined")
	})

	t.Run("pointer type", func(t *testing.T) {
		keys := knownJSONKeys(reflect.TypeFor[*GitOpsControls]())
		assert.Contains(t, keys, "macos_updates")
		assert.Contains(t, keys, "scripts")
		// From BaseItem
		assert.Contains(t, keys, "path")
	})

	t.Run("renameto alias accepted", func(t *testing.T) {
		// PolicySpec has `json:"team" renameto:"fleet"` — both should be known
		keys := knownJSONKeys(reflect.TypeFor[fleet.PolicySpec]())
		assert.Contains(t, keys, "team")
		assert.Contains(t, keys, "fleet")

		// LabelSpec has `json:"team_id" renameto:"fleet_id"`
		keys = knownJSONKeys(reflect.TypeFor[fleet.LabelSpec]())
		assert.Contains(t, keys, "team_id")
		assert.Contains(t, keys, "fleet_id")
	})

	t.Run("caching works", func(t *testing.T) {
		t1 := reflect.TypeFor[Label]()
		keys1 := knownJSONKeys(t1)
		keys2 := knownJSONKeys(t1)
		// Should return the same map (pointer equality after caching)
		assert.Equal(t, keys1, keys2)
	})
}

func TestValidateUnknownKeys(t *testing.T) {
	t.Parallel()

	t.Run("no unknown keys", func(t *testing.T) {
		data := map[string]any{
			"name":        "test-query",
			"query":       "SELECT 1",
			"description": "a test",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.QuerySpec](), []string{"reports", "[0]"}, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("unknown key detected", func(t *testing.T) {
		data := map[string]any{
			"name":          "test-query",
			"query":         "SELECT 1",
			"unknown_field": "bad",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.QuerySpec](), []string{"reports", "[0]"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "unknown_field")
		assert.Contains(t, errs[0].Error(), "reports.[0]")
		assert.Contains(t, errs[0].Error(), "test.yml")

		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Equal(t, "unknown_field", unknownErr.Field)
	})

	t.Run("multiple unknown keys", func(t *testing.T) {
		data := map[string]any{
			"name": "test",
			"bad1": "x",
			"bad2": "y",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.QuerySpec](), []string{"reports"}, "test.yml")
		assert.Len(t, errs, 2)
	})

	t.Run("api_key_json keys are not validated", func(t *testing.T) {
		// GoogleCalendarApiKey accepts the full Google service-account JSON blob,
		// so unknown-key validation is intentionally skipped for its contents.
		data := map[string]any{
			"google_calendar": []any{
				map[string]any{
					"domain": "example.com",
					"api_key_json": map[string]any{
						"client_email":                "test@example.com",
						"private_key":                 "some value",
						"type":                        "service_account",
						"project_id":                  "fleet-dogfood",
						"private_key_id":              "abc123",
						"client_id":                   "1234567890",
						"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
						"token_uri":                   "https://oauth2.googleapis.com/token",
						"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
						"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/foo",
						"universe_domain":             "googleapis.com",
					},
				},
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.Integrations](), []string{"org_settings", "integrations"}, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("unknown keys at google_calendar entry level are still rejected", func(t *testing.T) {
		// Skipping validation for api_key_json must not leak into siblings.
		data := map[string]any{
			"google_calendar": []any{
				map[string]any{
					"domain":       "example.com",
					"api_key_json": map[string]any{"client_email": "x", "private_key": "y"},
					"bad_sibling":  true,
				},
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.Integrations](), []string{"org_settings", "integrations"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "bad_sibling")
	})

	t.Run("scalar data no errors", func(t *testing.T) {
		errs := validateUnknownKeys("just a string", reflect.TypeFor[fleet.QuerySpec](), nil, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("nil data no errors", func(t *testing.T) {
		errs := validateUnknownKeys(nil, reflect.TypeFor[fleet.QuerySpec](), nil, "test.yml")
		assert.Empty(t, errs)
	})

	t.Run("nested struct validation", func(t *testing.T) {
		// MacOSSetup is a pointer-to-struct field in GitOpsControls
		data := map[string]any{
			"macos_setup": map[string]any{
				"bootstrap_package": "pkg.pkg",
				"bad_nested_field":  true,
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "bad_nested_field")
		assert.Contains(t, errs[0].Error(), "controls.macos_setup")
	})

	t.Run("slice validation", func(t *testing.T) {
		data := []any{
			map[string]any{
				"name":      "query1",
				"query":     "SELECT 1",
				"bad_field": "x",
			},
			map[string]any{
				"name":  "query2",
				"query": "SELECT 2",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[[]Query](), []string{"reports"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "bad_field")
	})

	t.Run("non-struct target type", func(t *testing.T) {
		data := map[string]any{
			"anything": "goes",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[string](), nil, "test.yml")
		assert.Empty(t, errs)
	})
}

func TestSuggestKey(t *testing.T) {
	t.Parallel()

	known := knownJSONKeys(reflect.TypeFor[fleet.QuerySpec]())

	t.Run("close typo suggests match", func(t *testing.T) {
		assert.Equal(t, "query", suggestKey("qurey", known))
		assert.Equal(t, "query", suggestKey("qeury", known))
		assert.Equal(t, "name", suggestKey("nme", known))
		assert.Equal(t, "interval", suggestKey("intervl", known))
		assert.Equal(t, "description", suggestKey("desciption", known))
	})

	t.Run("completely unrelated no suggestion", func(t *testing.T) {
		assert.Empty(t, suggestKey("zzzzzzzzz", known))
		assert.Empty(t, suggestKey("xylophone", known))
	})

	t.Run("suggestion included in error message", func(t *testing.T) {
		data := map[string]any{
			"name":  "q",
			"qurey": "SELECT 1",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.QuerySpec](), []string{"reports"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `did you mean "query"`)

		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Equal(t, "query", unknownErr.Suggestion)
	})

	t.Run("no suggestion when too distant", func(t *testing.T) {
		data := map[string]any{
			"name":      "q",
			"xylophone": "SELECT 1",
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[fleet.QuerySpec](), []string{"reports"}, "test.yml")
		require.Len(t, errs, 1)
		assert.NotContains(t, errs[0].Error(), "did you mean")

		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Empty(t, unknownErr.Suggestion)
	})

	t.Run("nested path recorded on error", func(t *testing.T) {
		data := map[string]any{
			"macos_updates": map[string]any{
				"deadlinee": "2024-01-01",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "team.yml")
		require.Len(t, errs, 1)

		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Equal(t, "controls.macos_updates", unknownErr.Path)
		assert.Equal(t, "deadlinee", unknownErr.Field)
		assert.Equal(t, "team.yml", unknownErr.Filename)
	})
}

func TestAnyFieldTypeRegistry(t *testing.T) {
	t.Parallel()

	t.Run("controls any-field recursion", func(t *testing.T) {
		data := map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "14.0",
				"deadline":        "2024-01-01",
				"deadlinee":       "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "deadlinee")
		assert.Contains(t, errs[0].Error(), "controls.macos_updates")
		assert.Contains(t, errs[0].Error(), `did you mean "deadline"`)
	})

	t.Run("windows_updates any-field recursion", func(t *testing.T) {
		data := map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":    5,
				"grace_period_das": 2, // typo
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "grace_period_das")
	})

	t.Run("macos_settings any-field recursion", func(t *testing.T) {
		data := map[string]any{
			"macos_settings": map[string]any{
				"custom_settings":  []any{},
				"custom_settingss": "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "custom_settingss")
	})

	t.Run("android_settings any-field recursion", func(t *testing.T) {
		data := map[string]any{
			"android_settings": map[string]any{
				"custom_settings": []any{},
				"certificatess":   "typo",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsControls](), []string{"controls"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "certificatess")
	})

	t.Run("all registered types present", func(t *testing.T) {
		overrides, ok := anyFieldTypes[reflect.TypeFor[GitOpsControls]()]
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

	t.Run("org_settings certificate_authorities any-field recursion", func(t *testing.T) {
		data := map[string]any{
			"certificate_authorities": map[string]any{
				"ndes_scep_proxy": map[string]any{},
				"digicert":        []any{},
				"unknown_ca_type": "bad",
			},
		}
		errs := validateUnknownKeys(data, reflect.TypeFor[GitOpsOrgSettings](), []string{"org_settings"}, "test.yml")
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "unknown_ca_type")
		assert.Contains(t, errs[0].Error(), "org_settings.certificate_authorities")
	})

	t.Run("org_settings registered types present", func(t *testing.T) {
		overrides, ok := anyFieldTypes[reflect.TypeFor[GitOpsOrgSettings]()]
		require.True(t, ok)
		assert.Contains(t, overrides, "certificate_authorities")
	})
}

func TestValidateRawKeys(t *testing.T) {
	t.Parallel()

	t.Run("valid json", func(t *testing.T) {
		raw := []byte(`{"name":"test","query":"SELECT 1"}`)
		errs := validateRawKeys(raw, reflect.TypeFor[fleet.QuerySpec](), "test.yml", []string{"reports"})
		assert.Empty(t, errs)
	})

	t.Run("unknown key in json", func(t *testing.T) {
		raw := []byte(`{"name":"test","query":"SELECT 1","typo_field":"bad"}`)
		errs := validateRawKeys(raw, reflect.TypeFor[fleet.QuerySpec](), "test.yml", []string{"reports"})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "typo_field")
	})

	t.Run("invalid json returns parse error", func(t *testing.T) {
		raw := []byte(`{invalid json`)
		errs := validateRawKeys(raw, reflect.TypeFor[fleet.QuerySpec](), "test.yml", []string{"reports"})
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "invalid")
	})

	t.Run("bool or", func(t *testing.T) {
		raw := []byte(`{"install_software": {"package_path": "./lib/ruby.yml"}}`)
		errs := validateRawKeys(raw, reflect.TypeFor[GitOpsPolicySpec](), "test.yml", []string{"policies"})
		assert.Empty(t, errs)
	})
}

func TestFilterWarnings(t *testing.T) {
	t.Parallel()

	t.Run("nil multierror", func(t *testing.T) {
		err := filterWarnings(nil, func(string, ...any) {}, reflect.TypeFor[*ParseUnknownKeyError]())
		assert.NoError(t, err)
	})

	t.Run("filters matching errors and logs them", func(t *testing.T) {
		multiError := &multierror.Error{}
		multiError = multierror.Append(multiError,
			&ParseUnknownKeyError{Filename: "test.yml", Field: "bad_key"},
			errors.New("some other error"),
			&ParseUnknownKeyError{Filename: "test.yml", Field: "another_bad"},
		)

		var warnings []string
		logFn := func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}

		result := filterWarnings(multiError, logFn, reflect.TypeFor[*ParseUnknownKeyError]())
		require.Error(t, result)
		assert.Contains(t, result.Error(), "some other error")
		assert.NotContains(t, result.Error(), "bad_key")
		assert.Len(t, warnings, 2)
		assert.Contains(t, warnings[0], "bad_key")
		assert.Contains(t, warnings[1], "another_bad")
	})

	t.Run("returns nil when all errors filtered", func(t *testing.T) {
		multiError := &multierror.Error{}
		multiError = multierror.Append(multiError,
			&ParseUnknownKeyError{Filename: "test.yml", Field: "bad1"},
			&ParseUnknownKeyError{Filename: "test.yml", Field: "bad2"},
		)

		result := filterWarnings(multiError, func(string, ...any) {}, reflect.TypeFor[*ParseUnknownKeyError]())
		assert.NoError(t, result)
	})

	t.Run("preserves all errors when none match", func(t *testing.T) {
		multiError := &multierror.Error{}
		multiError = multierror.Append(multiError,
			errors.New("error one"),
			errors.New("error two"),
		)

		result := filterWarnings(multiError, func(string, ...any) {}, reflect.TypeFor[*ParseUnknownKeyError]())
		require.Error(t, result)
		var resultMulti *multierror.Error
		require.True(t, errors.As(result, &resultMulti))
		assert.Len(t, resultMulti.Errors, 2)
	})

	t.Run("multiple filter types", func(t *testing.T) {
		multiError := &multierror.Error{}
		multiError = multierror.Append(multiError,
			&ParseUnknownKeyError{Filename: "test.yml", Field: "bad"},
			&ParseTypeError{Filename: "test.yml", Keys: []string{"controls"}},
			errors.New("kept error"),
		)

		result := filterWarnings(multiError, func(string, ...any) {},
			reflect.TypeFor[*ParseUnknownKeyError](),
			reflect.TypeFor[*ParseTypeError](),
		)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "kept error")
		assert.NotContains(t, result.Error(), "bad")
	})
}
