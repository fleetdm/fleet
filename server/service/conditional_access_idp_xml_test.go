package service

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

// renderConditionalAccessProfile executes the profile template directly, so these
// tests cover the template and its escaping without the surrounding service setup.
func renderConditionalAccessProfile(t *testing.T, d appleProfileTemplateData) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, conditionalAccessAppleProfileTemplateParsed.Execute(&buf, d))
	return buf.String()
}

// conditionalAccessPayload parses the rendered profile and returns the payload of the given type, so assertions can be
// made against the values a device would apply rather than against the escaped markup.
func conditionalAccessPayload(t *testing.T, profile, payloadType string) map[string]any {
	t.Helper()
	var root struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	_, err := plist.Unmarshal([]byte(profile), &root)
	require.NoError(t, err, "rendered profile must be a well-formed plist")

	for _, p := range root.PayloadContent {
		if p["PayloadType"] == payloadType {
			return p
		}
	}
	t.Fatalf("no %s payload found in profile", payloadType)
	return nil
}

func conditionalAccessProfileFixture() appleProfileTemplateData {
	return appleProfileTemplateData{
		CACertBase64:     "dGVzdC1jZXJ0aWZpY2F0ZQ==",
		CertificateCN:    "fleet-conditional-access",
		MTLSURL:          "https://fleet.example.com/mtls",
		CACertUUID:       "00000000-0000-0000-0000-000000000001",
		SCEPPayloadUUID:  "00000000-0000-0000-0000-000000000002",
		IdentityPrefUUID: "00000000-0000-0000-0000-000000000003",
		ChromeConfigUUID: "00000000-0000-0000-0000-000000000004",
		RootPayloadUUID:  "00000000-0000-0000-0000-000000000005",
	}
}

// TestConditionalAccessProfileEscapesInterpolatedValues checks that a value
// carrying XML markup becomes literal text rather than structure: the parsed
// profile gains no keys, and the value survives intact.
func TestConditionalAccessProfileEscapesInterpolatedValues(t *testing.T) {
	t.Parallel()

	payload := `https://okta.example.com</string><key>Injected</key><string>true</string><key>Ignored</key><string>`
	d := conditionalAccessProfileFixture()
	d.MTLSURL = payload

	pref := conditionalAccessPayload(t, renderConditionalAccessProfile(t, d), "com.apple.security.identitypreference")

	require.NotContains(t, pref, "Injected", "an interpolated value must not add keys to the payload")
	require.NotContains(t, pref, "Ignored")
	require.Equal(t, payload, pref["Name"], "the value must survive as literal text")
}

// TestConditionalAccessProfileChromePolicyIsValidJSON covers the one payload that
// carries stringified JSON inside an XML text node. XML escaping is the wrong
// layer there: a quote escaped as &#34; decodes back to " before Chrome parses
// the value, so a quote-bearing URL would break out of the JSON string.
func TestConditionalAccessProfileChromePolicyIsValidJSON(t *testing.T) {
	t.Parallel()

	d := conditionalAccessProfileFixture()
	d.MTLSURL = `https://fleet.example.com/mtls?x=","filter":{"SUBJECT":{"CN":"HIJACKED`

	var root struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	_, err := plist.Unmarshal([]byte(renderConditionalAccessProfile(t, d)), &root)
	require.NoError(t, err)

	var policy string
	for _, p := range root.PayloadContent {
		if p["PayloadType"] != "com.apple.ManagedClient.preferences" {
			continue
		}
		chrome := p["PayloadContent"].(map[string]any)["com.google.Chrome"].(map[string]any)
		forced := chrome["Forced"].([]any)
		settings := forced[0].(map[string]any)["mcx_preference_settings"].(map[string]any)
		policy = settings["AutoSelectCertificateForUrls"].([]any)[0].(string)
	}
	require.NotEmpty(t, policy, "Chrome auto-select policy should be present")

	var decoded struct {
		Pattern string `json:"pattern"`
		Filter  struct {
			Subject struct {
				CN string `json:"CN"`
			} `json:"SUBJECT"`
		} `json:"filter"`
	}
	require.NoError(t, json.Unmarshal([]byte(policy), &decoded),
		"policy must stay valid JSON: %s", policy)
	require.Equal(t, d.MTLSURL, decoded.Pattern, "URL must survive as a literal JSON string value")
	require.Equal(t, d.CertificateCN, decoded.Filter.Subject.CN,
		"a URL must not be able to overwrite the certificate filter")
}

// TestConditionalAccessProfileEscapesEveryField is a drift guard: it walks the
// template data by reflection and fails if any field reaches the profile
// unescaped. A field added later that forgets its escaping func fails here
// rather than shipping an injectable interpolation.
func TestConditionalAccessProfileEscapesEveryField(t *testing.T) {
	t.Parallel()

	const marker = `<INJECTED-MARKER/>`
	// The template escapes as XML in most places and as JSON inside the Chrome
	// policy, so accept either encoding of the marker.
	escapedForms := []string{`&lt;INJECTED-MARKER/&gt;`, `<INJECTED-MARKER/>`}

	fields := reflect.TypeFor[appleProfileTemplateData]()
	require.Positive(t, fields.NumField())

	for i := range fields.NumField() {
		f := fields.Field(i)
		require.Equal(t, reflect.String, f.Type.Kind(),
			"%s is not a string; this guard only covers string fields and needs updating", f.Name)

		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()

			d := conditionalAccessProfileFixture()
			reflect.ValueOf(&d).Elem().Field(i).SetString(marker)
			got := renderConditionalAccessProfile(t, d)

			require.NotContains(t, got, marker,
				"%s reaches the profile unescaped; wrap its interpolation with an escaping func", f.Name)

			var found bool
			for _, form := range escapedForms {
				if strings.Contains(got, form) {
					found = true
					break
				}
			}
			require.True(t, found,
				"%s never appears in the profile: either it is unused and should be removed from the struct, "+
					"or its interpolation is missing from the template", f.Name)
		})
	}
}
