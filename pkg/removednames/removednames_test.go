package removednames

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	t.Parallel()

	const suffix = " See " + GuideURL
	cases := []struct {
		name    string
		kind    Kind
		old     string
		context string
		want    string
	}{
		{"json key", KindJSONKey, "team_id", "", "team_id was removed in Fleet 5. Use fleet_id instead." + suffix},
		{"yaml key", KindYAMLKey, "team_settings", "", "team_settings was removed in Fleet 5. Use settings instead." + suffix},
		{"flag adds dashes", KindFlag, "team", "fleetctl get hosts", "--team was removed in Fleet 5. Use --fleet instead." + suffix},
		{"flag value", KindFlagValue, "--team no-team", "fleetctl generate-gitops",
			"--team no-team was removed in Fleet 5. Use --fleet unassigned instead." + suffix},
		{"env var", KindEnvVar, "DELETE_OTHER_TEAMS", "fleetctl gitops",
			"DELETE_OTHER_TEAMS was removed in Fleet 5. Use DELETE_OTHER_FLEETS instead." + suffix},
		{"path", KindPath, "/api/_version_/fleet/teams", "",
			"/api/_version_/fleet/teams was removed in Fleet 5. Use /api/_version_/fleet/fleets instead." + suffix},
		{"command", KindCommand, "fleetctl get teams", "",
			"fleetctl get teams was removed in Fleet 5. Use fleetctl get fleets instead." + suffix},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, ok := Find(c.kind, c.old, c.context)
			require.True(t, ok)
			assert.Equal(t, c.want, e.Message())
		})
	}

	assert.Equal(t, "a was removed in Fleet 5. Use b instead. See "+GuideURL, Message("a", "b"))
}

func TestFindAndFilter(t *testing.T) {
	t.Parallel()

	// A flag entry is tied to its command.
	_, ok := Find(KindFlag, "team", "fleetctl get hosts")
	assert.True(t, ok)
	_, ok = Find(KindFlag, "team", "fleetctl gitops")
	assert.False(t, ok)
	_, ok = Find(KindFlag, "policies-team", "fleetctl apply")
	assert.True(t, ok)

	// An entry without context matches every context.
	_, ok = Find(KindJSONKey, "team_id", "anything")
	assert.True(t, ok)

	// Scope filtering: activity details keys do not leak into the request scope and vice versa.
	for _, e := range Filter(KindJSONKey, ScopeRequest) {
		assert.Zero(t, e.Scope&ScopeActivityDetails, e.Old)
	}
	activityKeys := Filter(KindJSONKey, ScopeActivityDetails)
	assert.Len(t, activityKeys, 3)
	assert.Greater(t, len(Filter(KindJSONKey, 0)), len(activityKeys))

	assert.Len(t, All(), len(entries))
}

func TestEntriesWellFormed(t *testing.T) {
	t.Parallel()

	type id struct {
		kind    Kind
		scope   Scope
		old     string
		context string
	}
	seen := map[id]struct{}{}
	for _, e := range entries {
		assert.NotEmpty(t, e.Kind)
		assert.NotZero(t, e.Scope, e.Old)
		assert.NotEmpty(t, e.New, e.Old)
		assert.NotEqual(t, e.Old, e.New)
		assert.Contains(t, strings.ToLower(e.Old), "team", "old name should be team-related")

		key := id{e.Kind, e.Scope, e.Old, e.Context}
		_, dup := seen[key]
		assert.False(t, dup, "duplicate entry for %v", key)
		seen[key] = struct{}{}
	}
}

// TestEntriesCoverDeprecatedSources fails when a source that used to accept a team-related name as a deprecated alias has one that
// is missing from the table, so no old-to-new pair can be lost when those sources are removed.
func TestEntriesCoverDeprecatedSources(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	has := func(kind Kind, scope Scope, oldName, newName string) bool {
		return slices.ContainsFunc(entries, func(e Entry) bool {
			return e.Kind == kind && e.Scope&scope != 0 && e.Old == oldName && (newName == "" || e.New == newName)
		})
	}

	t.Run("renameto struct tags", func(t *testing.T) {
		t.Parallel()
		for rel, file := range parseFiles(t, root, []string{"server", "ee", "cmd", "pkg"}, "renameto") {
			ast.Inspect(file, func(n ast.Node) bool {
				field, ok := n.(*ast.Field)
				if !ok || field.Tag == nil {
					return true
				}
				unquoted, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					return true
				}
				tag := reflect.StructTag(unquoted)
				renameTo, _, _ := strings.Cut(tag.Get("renameto"), ",")
				if renameTo == "" {
					return true
				}
				for key, kind := range map[string]Kind{
					"json": KindJSONKey, "query": KindQueryParam, "url": KindQueryParam, "form": KindFormField, "csv": KindCSVColumn,
				} {
					oldName, _, _ := strings.Cut(tag.Get(key), ",")
					if oldName == "" || !hasTeamToken(oldName) {
						continue
					}
					scope := ScopeRequest | ScopeResponse
					if filepath.ToSlash(rel) == "server/fleet/activities.go" {
						scope = ScopeActivityDetails
					}
					assert.True(t, has(kind, scope, oldName, renameTo), "%s: %s tag %q renamed to %q is missing from entries", rel, key, oldName, renameTo)
				}
				return true
			})
		}
	})

	t.Run("deprecated team paths", func(t *testing.T) {
		t.Parallel()
		rel := filepath.Join("server", "service", "handler_deprecated_paths.go")
		file := parseFile(t, filepath.Join(root, rel))
		teamPath := regexp.MustCompile(`/teams?(/|$)`)
		paramRegex := regexp.MustCompile(`\{(\w+):[^}]*\}`)
		const prefix = "/api/_version_/fleet"
		found := 0
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			var primary string
			var deprecated []string
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				switch exprName(kv.Key) {
				case "PrimaryPath":
					primary = stringLit(kv.Value)
				case "DeprecatedPaths":
					deprecated = stringLits(kv.Value)
				}
			}
			for _, d := range deprecated {
				if !teamPath.MatchString(d) {
					continue
				}
				found++
				oldPath, newPath := paramRegex.ReplaceAllString(d, "{$1}"), paramRegex.ReplaceAllString(primary, "{$1}")
				assert.True(t, strings.HasPrefix(oldPath, prefix) && has(KindPath, ScopeRequest, oldPath, newPath),
					"deprecated path %s -> %s is missing from entries", oldPath, newPath)
			}
			return true
		})
		assert.Positive(t, found, "expected to find deprecated team paths in %s", rel)
	})

	t.Run("fleetctl flag, command, and env var aliases", func(t *testing.T) {
		t.Parallel()
		endsWithAny := func(context string, names []string) bool {
			return slices.ContainsFunc(names, func(name string) bool { return strings.HasSuffix(context, " "+name) })
		}
		found := 0
		for rel, file := range parseFiles(t, root, []string{filepath.Join("cmd", "fleetctl", "fleetctl")}, "") {
			for _, alias := range collectCLIAliases(file) {
				found++
				switch alias.kind {
				case cliCommandAlias:
					primary := alias.command[0]
					assert.True(t, slices.ContainsFunc(entries, func(e Entry) bool {
						return e.Kind == KindCommand && strings.HasSuffix(e.Old, " "+alias.name) && strings.HasSuffix(e.New, " "+primary)
					}), "%s: command alias %q of %q is missing from entries", rel, alias.name, primary)
				case cliFlagAlias:
					assert.True(t, slices.ContainsFunc(entries, func(e Entry) bool {
						return e.Kind == KindFlag && e.Old == alias.name && endsWithAny(e.Context, alias.command) &&
							(alias.flagName == "" || e.New == alias.flagName)
					}), "%s: flag alias %q of command %v is missing from entries", rel, alias.name, alias.command)
				case cliEnvVar:
					assert.True(t, slices.ContainsFunc(entries, func(e Entry) bool {
						return e.Kind == KindEnvVar && e.Old == alias.name && endsWithAny(e.Context, alias.command)
					}), "%s: env var %q of command %v is missing from entries", rel, alias.name, alias.command)
				}
			}
		}
		assert.Positive(t, found)
	})

	t.Run("gitops key mappings", func(t *testing.T) {
		t.Parallel()
		file := parseFile(t, filepath.Join(root, "pkg", "spec", "gitops_deprecations.go"))
		found := 0
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || len(lit.Elts) != 2 {
				return true
			}
			oldPath, newPath := stringLit(lit.Elts[0]), stringLit(lit.Elts[1])
			if oldPath == "" || newPath == "" || !hasTeamToken(oldPath) {
				return true
			}
			found++
			// A dotted path must have its own YAML entry; only an undotted name can be covered by a JSON key entry.
			matched := has(KindYAMLKey, ScopeRequest, oldPath, newPath)
			if !strings.Contains(oldPath, ".") {
				matched = matched || has(KindJSONKey, ScopeRequest, oldPath, newPath)
			}
			assert.True(t, matched, "gitops mapping %s -> %s is missing from entries", oldPath, newPath)
			return true
		})
		assert.Positive(t, found)
	})
}

var camelToken = regexp.MustCompile(`[A-Z][a-z0-9]*|[a-z0-9]+`)

// hasTeamToken reports whether name contains "team" or "teams" as a token in snake_case, kebab-case, camelCase, or dotted form.
func hasTeamToken(name string) bool {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == '[' || r == ']' })
	for _, part := range parts {
		tokens := camelToken.FindAllString(part, -1)
		if part == strings.ToUpper(part) {
			tokens = []string{part} // SCREAMING_CASE parts are whole tokens, not one token per letter
		}
		for _, token := range tokens {
			if lower := strings.ToLower(token); lower == "team" || lower == "teams" {
				return true
			}
		}
	}
	return false
}

// parseFiles parses the non-test Go files under dirs (relative to root) that contain substr, keyed by path relative to root.
func parseFiles(t *testing.T, root string, dirs []string, substr string) map[string]*ast.File {
	t.Helper()
	files := map[string]*ast.File{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == "testdata" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path) //nolint:gosec // the test reads the repo's own source files
			if err != nil {
				return err
			}
			if !strings.Contains(string(b), substr) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[rel] = parseFile(t, path)
			return nil
		})
		require.NoError(t, err)
	}
	return files
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	return file
}

func exprName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func stringLit(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return s
}

func stringLits(e ast.Expr) []string {
	switch v := e.(type) {
	case *ast.CompositeLit:
		var out []string
		for _, elt := range v.Elts {
			if s := stringLit(elt); s != "" {
				out = append(out, s)
			}
		}
		return out
	case *ast.BasicLit:
		if s := stringLit(v); s != "" {
			return []string{s}
		}
	}
	return nil
}

const (
	cliCommandAlias = "command alias"
	cliFlagAlias    = "flag alias"
	cliEnvVar       = "env var"
)

// cliAlias is a team-named command alias, flag alias, or env var found in a fleetctl command definition.
type cliAlias struct {
	kind string
	name string
	// command holds the name of the enclosing command followed by its aliases.
	command []string
	// flagName is the flag's name when it is a string literal.
	flagName string
}

// collectCLIAliases returns the team-named aliases and env vars declared in file, with the command that declares each.
func collectCLIAliases(file *ast.File) []cliAlias {
	var out []cliAlias
	var walk func(node ast.Node, command []string)
	walk = func(node ast.Node, command []string) {
		ast.Inspect(node, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Command" {
				names := slices.Concat(literalsOf(lit, "Name"), literalsOf(lit, "Aliases"))
				if len(names) > 0 {
					for _, alias := range literalsOf(lit, "Aliases") {
						if hasTeamToken(alias) {
							out = append(out, cliAlias{kind: cliCommandAlias, name: alias, command: names})
						}
					}
					for _, elt := range lit.Elts {
						walk(elt, names)
					}
					return false
				}
			}
			flagName := ""
			if names := literalsOf(lit, "Name"); len(names) == 1 {
				flagName = names[0]
			}
			for _, alias := range literalsOf(lit, "Aliases") {
				if hasTeamToken(alias) {
					out = append(out, cliAlias{kind: cliFlagAlias, name: alias, command: command, flagName: flagName})
				}
			}
			for _, envVar := range literalsOf(lit, "EnvVars") {
				if hasTeamToken(envVar) {
					out = append(out, cliAlias{kind: cliEnvVar, name: envVar, command: command})
				}
			}
			return true
		})
	}
	walk(file, nil)
	return out
}

// literalsOf returns the string literals assigned to the given key of a composite literal.
func literalsOf(lit *ast.CompositeLit, key string) []string {
	var out []string
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok && exprName(kv.Key) == key {
			out = append(out, stringLits(kv.Value)...)
		}
	}
	return out
}
