package fleetctl

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fleet-gitops skill that `fleetctl new` scaffolds into customer repos is a
// copy of the one in this repo's .claude/skills, placed at .agents/skills (the
// cross-agent location Codex, Cursor, Copilot, Gemini CLI, and Kilo read), with
// a thin .claude/skills entry so Claude Code finds it too. Customers only ever
// see the template copy, so a change to one that isn't mirrored is a bug.
func TestNewFleetGitopsSkillMatchesRepoSkill(t *testing.T) {
	repoSkill := filepath.Join("..", "..", "..", ".claude", "skills", "fleet-gitops")
	templateSkill := filepath.Join("templates", "new", ".agents", "skills", "fleet-gitops")
	claudeShim := filepath.Join("templates", "new", ".claude", "skills", "fleet-gitops")

	repoFiles := skillFiles(t, repoSkill)
	templateFiles := skillFiles(t, templateSkill)
	require.Equal(t, slices.Sorted(maps.Keys(repoFiles)), slices.Sorted(maps.Keys(templateFiles)),
		"file list differs between %s and %s", repoSkill, templateSkill)
	for rel, content := range repoFiles {
		require.Equal(t, string(content), string(templateFiles[rel]),
			"%s differs between %s and %s; copy the repo skill over the template", rel, repoSkill, templateSkill)
	}

	// The shim must trigger like the real skill (same name and description) and
	// hand off to it rather than carrying its own copy of the instructions.
	shimFiles := skillFiles(t, claudeShim)
	require.Equal(t, []string{"SKILL.md"}, slices.Sorted(maps.Keys(shimFiles)), "%s should hold only a SKILL.md that points at .agents/skills", claudeShim)
	shim := string(shimFiles["SKILL.md"])
	canonical := string(repoFiles["SKILL.md"])
	for _, key := range []string{"name:", "description:", "allowed-tools:"} {
		require.Equal(t, frontmatterLine(t, canonical, key), frontmatterLine(t, shim, key), "shim frontmatter %q differs from the skill's", key)
	}
	require.Contains(t, shim, ".agents/skills/fleet-gitops/SKILL.md")
}

func frontmatterLine(t *testing.T, skill, key string) string {
	t.Helper()
	parts := strings.SplitN(skill, "\n---\n", 2)
	require.Len(t, parts, 2, "SKILL.md has no closing frontmatter delimiter")
	for _, line := range strings.Split(parts[0], "\n") {
		if strings.HasPrefix(line, key) {
			return line
		}
	}
	t.Fatalf("frontmatter has no %q line", key)
	return ""
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
		content, err := os.ReadFile(path) //nolint:gosec // G122/G304: reading fixed files under the source tree in a test
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
