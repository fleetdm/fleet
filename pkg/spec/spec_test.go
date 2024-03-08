package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYaml(t *testing.T) {
	in := `
---
- Document
#---
--- Document2
---
Document3
`

	docs := SplitYaml(in)
	require.Equal(t, 3, len(docs))
	assert.Equal(t, "- Document\n#---", docs[0])
	assert.Equal(t, "Document2", docs[1])
	assert.Equal(t, "Document3", docs[2])
}

func gitRootPath(t *testing.T) string {
	path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(path))
}

func loadSpec(t *testing.T, relativePaths ...string) []byte {
	b, err := os.ReadFile(filepath.Join(
		append([]string{gitRootPath(t)}, relativePaths...)...,
	))
	require.NoError(t, err)
	return b
}

func TestGroupFromBytesWithStdLib(t *testing.T) {
	stdQueryLib := loadSpec(t,
		"docs", "01-Using-Fleet", "standard-query-library", "standard-query-library.yml",
	)
	g, err := GroupFromBytes(stdQueryLib)
	require.NoError(t, err)
	require.NotEmpty(t, g.Queries)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesWithMacOS13CISQueries(t *testing.T) {
	cisQueries := loadSpec(t,
		"ee", "cis", "macos-13", "cis-policy-queries.yml",
	)
	g, err := GroupFromBytes(cisQueries)
	require.NoError(t, err)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesWithWin10CISQueries(t *testing.T) {
	cisQueries := loadSpec(t,
		"ee", "cis", "win-10", "cis-policy-queries.yml",
	)
	g, err := GroupFromBytes(cisQueries)
	require.NoError(t, err)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesMissingFields(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{
			"missing spec",
			[]byte(`
---
apiVersion: v1
kind: team
			`),
			`Missing required fields ("spec") on provided "team" configuration.`,
		},
		{
			"missing spec and kind",
			[]byte(`
---
apiVersion: v1
			`),
			`Missing required fields ("spec", "kind") on provided configuration`,
		},
		{
			"missing spec and empty string kind",
			[]byte(`
---
apiVersion: v1
kind: ""
			`),
			`Missing required fields ("spec", "kind") on provided configuration`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GroupFromBytes(tt.in)
			require.ErrorContains(t, err, tt.want)
		})
	}
}
