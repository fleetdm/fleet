package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOutput(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}

	got, err := readOutput(filepath.Join(dir, "missing.json"))
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = readOutput(write("empty.json", ""))
	require.NoError(t, err)
	assert.Nil(t, got)

	// Only the header matters; the rest of the document isn't decoded.
	valid := `{"format_version": 1, "version": {"apple": {"release": "Release-v27.0", "commit": "` + testSHA + `"},
		"generated_at": "2026-10-15T00:00:00Z"}, "apple_profiles": "not checked here"}`
	got, err = readOutput(write("valid.json", valid))
	require.NoError(t, err)
	assert.Equal(t, &existingOutput{
		Version: profilevalidation.Version{
			Apple:       profilevalidation.AppleVersion{Release: "Release-v27.0", Commit: testSHA},
			GeneratedAt: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		},
		Data: []byte(valid),
	}, got)

	_, err = readOutput(write("bad.json", "not json"))
	require.ErrorContains(t, err, "read version from")
}

// validSchema returns a schema that passes profilevalidation.Parse: it sits on the floors and has
// every must-have entry.
func validSchema() *profilevalidation.ProfileValidationSchema {
	str := profilevalidation.AppleTypes{profilevalidation.AppleTypeString}
	s := &profilevalidation.ProfileValidationSchema{
		Version: profilevalidation.Version{Apple: profilevalidation.AppleVersion{Release: "Release-v27.0", Commit: testSHA}},
		AppleEnvelope: profilevalidation.AppleEnvelope{
			TopLevel:          &profilevalidation.ApplePayloadType{Source: "apple"},
			CommonPayloadKeys: &profilevalidation.ApplePayloadType{Source: "apple"},
		},
		AppleDeclarationEnvelope: &profilevalidation.ApplePayloadType{Source: "apple"},
		AppleProfiles: map[string]*profilevalidation.ApplePayloadType{
			"com.apple.applicationaccess": {Source: "apple", Keys: map[string]*profilevalidation.AppleKey{"allowCamera": {Type: str}}},
			// Characters json.Marshal would HTML-escape by default.
			"com.apple.eas.account": {Source: "apple", Keys: map[string]*profilevalidation.AppleKey{"Host": {Type: str, Format: "^<a&b>$"}}},
		},
		AppleDeclarations: map[string]*profilevalidation.ApplePayloadType{
			"com.apple.configuration.passcode.settings": {Source: "apple"},
		},
		Windows: map[string]*profilevalidation.WindowsNode{
			"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": {Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormatInt}, Source: profilevalidation.WindowsSourceWindows},
		},
		WindowsXSDs: map[string]string{},
	}
	for i := len(s.Windows); i < 4000; i++ {
		s.Windows[fmt.Sprintf("./Device/Vendor/MSFT/Example/Node%04d", i)] = &profilevalidation.WindowsNode{Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormatChr}, Source: profilevalidation.WindowsSourceWindows}
	}
	for i := len(s.AppleProfiles); i < 100; i++ {
		s.AppleProfiles[fmt.Sprintf("com.example.p%03d", i)] = &profilevalidation.ApplePayloadType{Source: "apple"}
	}
	for i := len(s.AppleDeclarations); i < 40; i++ {
		s.AppleDeclarations[fmt.Sprintf("com.apple.configuration.example%02d", i)] = &profilevalidation.ApplePayloadType{Source: "apple"}
	}
	return s
}

func TestWriteOutput(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	first := time.Date(2026, 10, 15, 10, 30, 45, 999, time.UTC)
	later := first.Add(48 * time.Hour)

	dir := t.TempDir()
	out := filepath.Join(dir, "profile_schema.json")

	// No previous output: written, timestamped, parseable.
	require.NoError(t, writeOutput(t.Context(), logger, out, validSchema(), nil, first))
	written, err := os.ReadFile(out)
	require.NoError(t, err)
	parsed, err := profilevalidation.Parse(written)
	require.NoError(t, err)
	assert.Equal(t, profilevalidation.FormatVersion, parsed.FormatVersion)
	assert.Equal(t, first.Truncate(time.Second), parsed.Version.GeneratedAt, "timestamp is UTC, to the second")
	assert.Contains(t, string(written), `"format": "^<a&b>$"`, "regexes are not HTML-escaped")
	assert.Contains(t, string(written), "\n  \"apple_declaration_envelope\": {", "output is indented")
	assert.Equal(t, byte('\n'), written[len(written)-1])

	// Same content again, later: byte-identical, generated_at kept.
	previous, err := readOutput(out)
	require.NoError(t, err)
	require.NoError(t, writeOutput(t.Context(), logger, out, validSchema(), previous, later))
	again, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, written, again)

	// Changed content: generated_at moves.
	changed := validSchema()
	changed.Version.Apple = profilevalidation.AppleVersion{Release: "Release-v28.0", Commit: "1111111111111111111111111111111111111111"}
	require.NoError(t, writeOutput(t.Context(), logger, out, changed, previous, later))
	got, err := readOutput(out)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, later.Truncate(time.Second), got.Version.GeneratedAt)
	assert.Equal(t, "Release-v28.0", got.Version.Apple.Release)

	assertNoTempFiles(t, dir)
}

func TestWriteOutputDeterministic(t *testing.T) {
	// Map iteration order is random; encoding must not be.
	a, err := encodeSchema(validSchema())
	require.NoError(t, err)
	for range 5 {
		b, err := encodeSchema(validSchema())
		require.NoError(t, err)
		require.Equal(t, a, b)
	}
}

func TestWriteOutputRejectsShrunkenSchema(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	shrunk := validSchema()
	for name := range shrunk.AppleProfiles {
		if name != "com.apple.applicationaccess" && len(shrunk.AppleProfiles) > 50 {
			delete(shrunk.AppleProfiles, name)
		}
	}

	t.Run("existing file left untouched", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "profile_schema.json")
		require.NoError(t, writeOutput(t.Context(), logger, out, validSchema(), nil, now))
		before, err := os.ReadFile(out)
		require.NoError(t, err)
		previous, err := readOutput(out)
		require.NoError(t, err)

		err = writeOutput(t.Context(), logger, out, shrunk, previous, now.Add(time.Hour))
		require.ErrorContains(t, err, "generated schema fails its own checks")
		require.ErrorContains(t, err, "apple_profiles has 50 payload types, want at least 100")
		after, err := os.ReadFile(out)
		require.NoError(t, err)
		assert.Equal(t, before, after)
		assertNoTempFiles(t, dir)
	})

	t.Run("no file created", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "profile_schema.json")
		require.Error(t, writeOutput(t.Context(), logger, out, shrunk, nil, now))
		_, err := os.Stat(out)
		require.ErrorIs(t, err, fs.ErrNotExist)
		assertNoTempFiles(t, dir)
	})
}

func TestWriteOutputMissingDir(t *testing.T) {
	err := writeOutput(t.Context(), slog.New(slog.DiscardHandler), filepath.Join(t.TempDir(), "missing", "out.json"), validSchema(), nil, time.Now())
	require.Error(t, err)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.Equal(t, "profile_schema.json", e.Name(), "unexpected file left behind")
	}
}
