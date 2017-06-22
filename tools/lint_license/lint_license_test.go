package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathFunctions(t *testing.T) {
	// Restore GOPATH after this test
	defer os.Setenv("GOPATH", os.Getenv("GOPATH"))

	gopath := "/gopath"
	os.Setenv("GOPATH", gopath)
	expectedPrefix := filepath.Join(gopath, "src/github.com/kolide/fleet")
	assert.Equal(t, filepath.Join(expectedPrefix, "foo/bar"), absolutePath("foo/bar"))

	assert.Equal(t, "foo/bar/baz", relativePath(absolutePath("foo/bar/baz")))
}

func TestCheckLicenses(t *testing.T) {
	config := settings{}
	deps := []dependency{}
	assert.Empty(t, checkLicenses(config, deps))

	// No allowed licenses
	deps = []dependency{
		{Name: "foobar", License: "MIT"},
	}
	assert.Equal(t, deps, checkLicenses(config, deps))

	// All (one) licenses acceptable
	config.AllowedLicenses = map[string]string{
		"MIT":       "fake_url",
		"Unlicense": "fake_url",
	}
	assert.Empty(t, checkLicenses(config, deps))

	// Some good, some bad
	deps = []dependency{
		{Name: "foobar", License: "MIT"},
		{Name: "bingle", License: ""},
		{Name: "quant", License: "Unlicense"},
		{Name: "grokk", License: "GPL2.0"},
	}
	invalid := checkLicenses(config, deps)
	if assert.Len(t, invalid, 2) {
		assert.Contains(t, invalid, deps[1])
		assert.Contains(t, invalid, deps[3])
	}
}

func TestWriteLicensesMarkdown(t *testing.T) {
	out := &bytes.Buffer{}
	config := settings{
		AllowedLicenses: map[string]string{
			"MIT":       "mit_url",
			"Unlicense": "unlicense_url",
			"Apache2.0": "apache_url",
		},
	}

	deps := []dependency{
		{Name: "foobar", SourceURL: "foobar_url", License: "MIT"},
		{Name: "quant", SourceURL: "quant_url", License: "Unlicense"},
		{Name: "zingle", SourceURL: "zingle_url", License: "Apache2.0"},
		{Name: "manx", SourceURL: "manx_url", License: "MIT"},
	}

	err := writeLicensesMarkdown(config, deps, out)
	require.Nil(t, err)

	md := out.String()

	expectedTableLine := func(dep dependency) string {
		return fmt.Sprintf("| [%s](%s) | [%s](%s) |",
			dep.Name, dep.SourceURL, dep.License, config.AllowedLicenses[dep.License])
	}
	for _, dep := range deps {
		assert.Contains(t, md, expectedTableLine(dep))
	}
}
