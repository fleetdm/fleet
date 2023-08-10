package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func TestFirstDuplicatePolicySpecName(t *testing.T) {
	t.Run("duplicate policy names", func(t *testing.T) {
		group := &Group{
			Policies: []*fleet.PolicySpec{
				{Name: "policy1"},
				{Name: "policy2"},
				{Name: "policy1"},
			},
		}

		policyName := group.FirstDuplicatePolicySpecName()

		require.NotEmpty(t, policyName)
		require.Equal(t, "policy1", policyName)
	})

	t.Run("no duplicate policy names", func(t *testing.T) {
		group := &Group{
			Policies: []*fleet.PolicySpec{
				{Name: "policy1"},
				{Name: "policy2"},
			},
		}

		policyName := group.FirstDuplicatePolicySpecName()

		require.Empty(t, policyName)
	})
}
