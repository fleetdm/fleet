package fleetctl

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fleet-gitops skill that `fleetctl new` scaffolds into customer repos is a
// copy of the one in this repo's .claude/skills. Customers only ever see the
// template copy, so a change to one that isn't mirrored to the other is a bug.
func TestNewFleetGitopsSkillMatchesRepoSkill(t *testing.T) {
	repoSkill := filepath.Join("..", "..", "..", ".claude", "skills", "fleet-gitops")
	templateSkill := filepath.Join("templates", "new", ".claude", "skills", "fleet-gitops")

	repoFiles := skillFiles(t, repoSkill)
	templateFiles := skillFiles(t, templateSkill)
	require.Equal(t, slices.Sorted(maps.Keys(repoFiles)), slices.Sorted(maps.Keys(templateFiles)),
		"file list differs between %s and %s", repoSkill, templateSkill)
	for rel, content := range repoFiles {
		require.Equal(t, string(content), string(templateFiles[rel]),
			"%s differs between %s and %s; copy the repo skill over the template", rel, repoSkill, templateSkill)
	}
}

func skillFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = content
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, files, "no files under %s", root)
	return files
}
