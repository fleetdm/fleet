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

func loadStdQueryLibrary(t *testing.T) []byte {
	b, err := os.ReadFile(filepath.Join(
		gitRootPath(t),
		"docs", "01-Using-Fleet", "standard-query-library", "standard-query-library.yml",
	))
	require.NoError(t, err)
	return b
}

func TestGroupFromBytes(t *testing.T) {
	stdQueryLib := loadStdQueryLibrary(t)
	g, err := GroupFromBytes(stdQueryLib)
	require.NoError(t, err)
	require.NotEmpty(t, g.Queries)
	require.NotEmpty(t, g.Policies)
}
