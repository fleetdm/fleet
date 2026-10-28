package profilevalidation

import (
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseMinimal(t *testing.T, mutate func(doc map[string]any)) *ProfileValidationSchema {
	t.Helper()
	doc := minimalDoc()
	if mutate != nil {
		mutate(doc)
	}
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	s, err := Parse(data)
	require.NoError(t, err)
	return s
}

// resetHolder clears the package-level holder for the test; these tests must not run in parallel.
func resetHolder(t *testing.T) {
	prevRaw, prev := profileValidationSchemaRaw, current.Load()
	current.Store(nil)
	t.Cleanup(func() {
		profileValidationSchemaRaw = prevRaw
		current.Store(prev)
	})
}

func TestParseProfileValidationSchema(t *testing.T) {
	resetHolder(t)

	profileValidationSchemaRaw = nil
	require.ErrorContains(t, ParseProfileValidationSchema(), "no embedded profile validation schema found")
	assert.Nil(t, Current())

	profileValidationSchemaRaw = []byte(`{"format_version": 2}`)
	require.ErrorContains(t, ParseProfileValidationSchema(), "unsupported profile validation schema format version: 2")
	assert.Nil(t, Current())

	data, err := json.Marshal(minimalDoc())
	require.NoError(t, err)
	profileValidationSchemaRaw = data
	require.NoError(t, ParseProfileValidationSchema())
	first := Current()
	require.NotNil(t, first)

	require.ErrorContains(t, ParseProfileValidationSchema(), "already parsed")
	assert.Same(t, first, Current())
}

func TestReplace(t *testing.T) {
	resetHolder(t)

	oldSchema := parseMinimal(t, nil)
	require.NoError(t, Replace(oldSchema))
	held := Current()
	require.Same(t, oldSchema, held)
	assert.Nil(t, held.AppleProfile("com.example.new"))

	newSchema := parseMinimal(t, func(doc map[string]any) {
		section(doc, "apple_profiles")["com.example.new"] = map[string]any{"source": "override"}
		section(doc, "version")["apple"] = map[string]any{"release": "Release-v28.0", "commit": "def"}
	})
	require.NoError(t, Replace(newSchema))

	// New lookups see the new schema.
	require.NotNil(t, Current().AppleProfile("com.example.new"))
	assert.Equal(t, "Release-v28.0", Current().Version.Apple.Release)

	// A schema handed out before the replace is unchanged.
	assert.Nil(t, held.AppleProfile("com.example.new"))
	assert.Equal(t, "Release-v27.0", held.Version.Apple.Release)

	require.Error(t, Replace(nil))
	assert.Same(t, newSchema, Current())

	require.ErrorContains(t, Replace(&ProfileValidationSchema{FormatVersion: 2}), "unexpected format version 2")
	require.ErrorContains(t, Replace(&ProfileValidationSchema{FormatVersion: FormatVersion}), "missing TopLevel or CommonPayloadKeys")
}

func TestReplaceConcurrentReads(t *testing.T) {
	resetHolder(t)
	a, b := parseMinimal(t, nil), parseMinimal(t, nil)
	require.NoError(t, Replace(a))

	// Meaningful under -race: readers and a writer touch the holder concurrently.
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 1000 {
				s := Current()
				assert.NotNil(t, s.AppleProfile("com.apple.applicationaccess"))
			}
		})
	}
	wg.Go(func() {
		for i := range 1000 {
			_ = Replace([]*ProfileValidationSchema{a, b}[i%2])
		}
	})
	wg.Wait()
}

func TestLookups(t *testing.T) {
	s := parseMinimal(t, nil)

	p := s.AppleProfile("com.apple.applicationaccess")
	require.NotNil(t, p)
	assert.Equal(t, AppleTypes{AppleTypeBoolean}, notNil(t, p.Keys["allowCamera"]).Type)
	assert.NotNil(t, s.AppleDeclaration("com.apple.configuration.passcode.settings"))

	assert.Nil(t, s.AppleProfile("com.example.unknown"))
	assert.Nil(t, s.AppleDeclaration("com.example.unknown"))
	// Profile and declaration names are separate namespaces.
	assert.Nil(t, s.AppleProfile("com.apple.configuration.passcode.settings"))
	assert.Nil(t, s.AppleDeclaration("com.apple.applicationaccess"))

	assert.NotNil(t, s.WindowsNode(mustHaveWindowsNodes[0]))
	assert.Nil(t, s.WindowsNode("./Device/Vendor/MSFT/Unknown"))
}

func TestNoServerFleetImport(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err)
	for dep := range strings.SplitSeq(string(out), "\n") {
		assert.False(t, strings.HasPrefix(dep, "github.com/fleetdm/fleet/v4/server/fleet"), "package must not depend on %s", dep)
	}
}
