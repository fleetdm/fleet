package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemplates(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "windows-admx-templates.json")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestAddWindowsTemplates(t *testing.T) {
	const content = `{"generatedAt": "2026-10-07T14:08:30.957Z", "templates": [{"id": "google-chrome"}], "nodes": [
		{"admxTemplate": "google-chrome", "locUri": "./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation",
		 "scopes": ["Device", "User"], "format": "chr", "accessType": "Add, Delete, Get, Replace", "admxElements": [], "description": "ignored"},
		{"admxTemplate": "google-chrome", "locUri": "./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome/DeviceOnly",
		 "scopes": ["Device"], "format": "chr", "deprecated": true},
		{"admxTemplate": "zoom", "locUri": "./Device/Vendor/MSFT/Policy/Config/ZoomMeetingsHKLM72~Policy~zoom/UserOnly",
		 "scopes": ["User"], "format": "chr"}
	]}`
	path := writeTemplates(t, content)
	ddf := &profilevalidation.WindowsNode{Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormatInt}, Source: profilevalidation.WindowsSourceWindows}
	schema := &profilevalidation.ProfileValidationSchema{Windows: map[string]*profilevalidation.WindowsNode{
		"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": ddf,
	}}

	got, err := addWindowsTemplates(t.Context(), slog.New(slog.DiscardHandler), path, schema)
	require.NoError(t, err)
	assert.Equal(t, profilevalidation.WindowsTemplatesVersion{SHA256: sha256Hex([]byte(content)), GeneratedAt: "2026-10-07T14:08:30.957Z"}, got)

	chr := profilevalidation.WindowsFormats{profilevalidation.WindowsFormatChr}
	tmpl := profilevalidation.WindowsSourceADMXTemplate
	assert.Equal(t, map[string]*profilevalidation.WindowsNode{
		"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock":                    ddf,
		"./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation": {Format: chr, Scope: profilevalidation.WindowsScopeDevice, Source: tmpl},
		"./User/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation":   {Format: chr, Scope: profilevalidation.WindowsScopeUser, Source: tmpl},
		"./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome/DeviceOnly": {
			Format: chr, Scope: profilevalidation.WindowsScopeDevice, Source: tmpl, Deprecated: &profilevalidation.WindowsDeprecated{},
		},
		"./User/Vendor/MSFT/Policy/Config/ZoomMeetingsHKLM72~Policy~zoom/UserOnly": {Format: chr, Scope: profilevalidation.WindowsScopeUser, Source: tmpl},
	}, schema.Windows)
}

func TestAddWindowsTemplatesErrors(t *testing.T) {
	node := func(fields string) string {
		return `{"generatedAt": "x", "nodes": [{"admxTemplate": "t", "format": "chr", ` + fields + `}]}`
	}
	const homepage = `"locUri": "./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation"`
	cases := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "outside Policy/Config", content: node(`"locUri": "./Device/Vendor/MSFT/Firewall/X", "scopes": ["Device"]`), wantErr: `policy "./Device/Vendor/MSFT/Firewall/X" isn't under ./Device/Vendor/MSFT/Policy/Config/`},
		{name: "user root", content: node(`"locUri": "./User/Vendor/MSFT/Policy/Config/A~Policy~b/C", "scopes": ["User"]`), wantErr: "isn't under"},
		{name: "no scopes", content: node(homepage), wantErr: "has no scopes"},
		{name: "unknown scope", content: node(homepage + `, "scopes": ["Machine"]`), wantErr: `has unknown scope "Machine"`},
		{name: "listed twice", content: `{"nodes": [{"format": "chr", ` + homepage + `, "scopes": ["Device"]}, {"format": "chr", ` + homepage + `, "scopes": ["Device"]}]}`, wantErr: "is already in the schema"},
		{name: "no policies", content: `{"generatedAt": "x", "nodes": []}`, wantErr: "has no policies"},
		{name: "invalid JSON", content: `{"nodes": [`, wantErr: "decode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			schema := &profilevalidation.ProfileValidationSchema{Windows: map[string]*profilevalidation.WindowsNode{}}
			_, err := addWindowsTemplates(t.Context(), slog.New(slog.DiscardHandler), writeTemplates(t, c.content), schema)
			require.ErrorContains(t, err, c.wantErr)
		})
	}

	t.Run("clash with a DDF node", func(t *testing.T) {
		schema := &profilevalidation.ProfileValidationSchema{Windows: map[string]*profilevalidation.WindowsNode{
			"./User/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation": {Source: profilevalidation.WindowsSourceWindows},
		}}
		_, err := addWindowsTemplates(t.Context(), slog.New(slog.DiscardHandler), writeTemplates(t, node(homepage+`, "scopes": ["Device", "User"]`)), schema)
		require.ErrorContains(t, err, "./User/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation is already in the schema")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := addWindowsTemplates(t.Context(), slog.New(slog.DiscardHandler), filepath.Join(t.TempDir(), "missing.json"), &profilevalidation.ProfileValidationSchema{})
		require.ErrorContains(t, err, "read ADMX template data")
	})
}
