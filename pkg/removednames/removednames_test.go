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
	for _, c := range []struct {
		kind    Kind
		old     string
		context string
		want    string
	}{
		// The Fleet 5 error spec: one wording for parameters and YAML keys, with the replacement left to the guide.
		{KindQueryParam, "team_id", "", `The "team_id" parameter is not available anymore. What to use instead: ` + GuideURL},
		{KindJSONKey, "team_id", "", `The "team_id" parameter is not available anymore. What to use instead: ` + GuideURL},
		{KindYAMLKey, "team_settings", "", `The "team_settings" key is not available anymore. What to use instead: ` + GuideURL},
		{KindSortKey, "team_id", "", `The "team_id" parameter is not available anymore. What to use instead: ` + GuideURL},
		{KindTargetType, "team", "", `The "team" parameter is not available anymore. What to use instead: ` + GuideURL},
		{KindCommand, "fleetctl get teams", "", `The "fleetctl get teams" command is not available anymore. What to use instead: ` + GuideURL},
		{KindEnvVar, "DELETE_OTHER_TEAMS", "fleetctl gitops",
			`The "DELETE_OTHER_TEAMS" environment variable is not available anymore. What to use instead: ` + GuideURL},
		{KindFlagValue, "--fleet no-team", "fleetctl generate-gitops",
			`The "--fleet no-team" value is not available anymore. What to use instead: ` + GuideURL},
		// Only flags are displayed differently: they get their leading dashes.
		{KindFlag, "team", "fleetctl get hosts", `The "--team" flag is not available anymore. What to use instead: ` + GuideURL},
		{KindSAMLAttribute, "FLEET_JIT_USER_ROLE_TEAM_<ID>", "",
			`The "FLEET_JIT_USER_ROLE_TEAM_<ID>" SAML attribute is not available anymore. What to use instead: ` + GuideURL},
		// A removed endpoint leaves the replacement to the guide. The request's method is added when a handler answers it.
		{KindPath, pathPrefix + "/spec/teams", "", `The "/spec/teams" endpoint is not available anymore. What to use instead: ` + GuideURL},
		{KindFlag, "t", "fleetctl user create", `The "-t" flag is not available anymore. What to use instead: ` + GuideURL},
	} {
		e, ok := Find(c.kind, c.old, c.context)
		require.True(t, ok)
		assert.Equal(t, c.want, e.Message())
	}

	assert.Equal(t, "a was removed in Fleet 5. Use b instead. See "+GuideURL, Message("a", "b"))
	assert.Equal(t, "a was removed in Fleet 5."+suffix, Message("a", ""))
}

func TestEntryMessageFormats(t *testing.T) {
	t.Parallel()

	const suffix = " See " + GuideURL
	for _, c := range []struct {
		name  string
		entry Entry
		want  string
	}{
		{"nil format is ReplacedFormat for a CSV column", Entry{Kind: KindCSVColumn, Old: "a", New: "b"},
			"a was removed in Fleet 5. Use b instead." + suffix},
		{"nil format is ParameterFormat for a JSON key", Entry{Kind: KindJSONKey, Old: "a", New: "b"},
			`The "a" parameter is not available anymore. What to use instead: ` + GuideURL},
		{"nil format is KeyFormat for a YAML key", Entry{Kind: KindYAMLKey, Old: "a.b", New: "a.c"},
			`The "a.b" key is not available anymore. What to use instead: ` + GuideURL},
		{"nil format is EndpointFormat for a path", Entry{Kind: KindPath, Old: pathPrefix + "/a", Context: "GET"},
			`The "GET /a" endpoint is not available anymore. What to use instead: ` + GuideURL},
		{"replaced without a replacement", Entry{Kind: KindJSONKey, Old: "a", Format: ReplacedFormat{}}, "a was removed in Fleet 5." + suffix},
		{"a flag with a replacement is not shown with it", Entry{Kind: KindFlag, Old: "abc", New: "def"},
			`The "--abc" flag is not available anymore. What to use instead: ` + GuideURL},
		{"replaced flag has dashes and no dangling replacement", Entry{Kind: KindFlag, Old: "a", Format: ReplacedFormat{}},
			"-a was removed in Fleet 5." + suffix},
		{
			"endpoint without a method",
			Entry{Kind: KindPath, Old: pathPrefix + "/x/{id}", Format: EndpointFormat{}},
			`The "/x/{id}" endpoint is not available anymore. What to use instead: ` + GuideURL,
		},
		{
			"endpoint with a method",
			Entry{Kind: KindPath, Old: pathPrefix + "/x", Context: "POST", Format: EndpointFormat{}},
			`The "POST /x" endpoint is not available anymore. What to use instead: ` + GuideURL,
		},
		{
			"endpoint that a setting can turn back on",
			Entry{Kind: KindPath, Old: pathPrefix + "/x", Context: "GET", Format: EndpointFormat{ByDefault: true}},
			`The "GET /x" endpoint is not available anymore by default. What to use instead: ` + GuideURL,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, c.entry.Message())
		})
	}
}

func TestFindAndFilter(t *testing.T) {
	t.Parallel()

	// A SAML attribute matches on the text before its <ID>.
	_, ok := FindSAMLAttribute("FLEET_JIT_USER_ROLE_TEAM_12")
	assert.True(t, ok)
	_, ok = FindSAMLAttribute("FLEET_JIT_USER_ROLE_FLEET_12")
	assert.False(t, ok)
	// A flag entry is tied to its command, and an entry without a context matches every context.
	_, ok = Find(KindFlag, "team", "fleetctl get hosts")
	assert.True(t, ok)
	_, ok = Find(KindFlag, "team", "fleetctl gitops")
	assert.False(t, ok)
	_, ok = Find(KindJSONKey, "team_id", "anything")
	assert.True(t, ok)

	// A YAML value ignores case, but its key does not.
	for name, want := range map[string]bool{
		"name: No team": true, "name: no team": true, "name: NO TEAM": true, "NAME: No team": false, "name: Unassigned": false, "No team": false,
	} {
		_, ok = Find(KindYAMLValue, name, "")
		assert.Equal(t, want, ok, name)
	}

	// Filter matches on scope bits, and a zero scope matches every scope.
	for _, scope := range []Scope{ScopeRequest, ScopeResponse, ScopeActivityDetails} {
		entries := Filter(KindJSONKey, scope)
		assert.NotEmpty(t, entries)
		for _, e := range entries {
			assert.Equal(t, KindJSONKey, e.Kind)
			assert.NotZero(t, e.Scope&scope, e.Old)
		}
	}
	assert.Greater(t, len(Filter(KindJSONKey, 0)), len(Filter(KindJSONKey, ScopeActivityDetails)))
}

func TestFindByContext(t *testing.T) {
	t.Parallel()

	list := []Entry{{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/x", Context: "GET"}}
	_, ok := findIn(list, KindPath, pathPrefix+"/x", "GET")
	assert.True(t, ok)
	_, ok = findIn(list, KindPath, pathPrefix+"/x", "POST")
	assert.False(t, ok)
}

func TestEntriesWellFormed(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, entries)

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
		assert.NotEqual(t, e.Old, e.New)
		if e.Kind == KindYAMLValue {
			assert.Contains(t, e.Old, ": ", "a YAML value is written key: value")
			assert.Contains(t, e.New, ": ", "a YAML value is written key: value")
		}
		assert.Contains(t, e.Message(), GuideURL, e.Old)
		switch e.Format.(type) {
		case nil:
		case ReplacedFormat:
			assert.NotEqual(t, KindPath, e.Kind, "a removed path uses the endpoint message: %s", e.Old)
		case ParameterFormat, KeyFormat:
		case NounFormat:
			assert.NotEqual(t, KindPath, e.Kind, "a removed path uses the endpoint message: %s", e.Old)
		case EndpointFormat:
			assert.Equal(t, KindPath, e.Kind, "an endpoint message is for a path: %s", e.Old)
		default:
			assert.Fail(t, "unknown format", e.Old)
		}
		if e.Kind == KindPath {
			assert.True(t, strings.HasPrefix(e.Old, pathPrefix), "path should start with %s: %s", pathPrefix, e.Old)
		}

		key := id{e.Kind, e.Scope, e.Old, e.Context}
		_, dup := seen[key]
		assert.False(t, dup, "duplicate entry for %v", key)
		seen[key] = struct{}{}
	}

	// Renaming teams always has a replacement, and every old name is team-related.
	for _, e := range teamEntries {
		assert.NotEmpty(t, e.New, e.Old)
		// The exceptions are the single-letter aliases of team commands and flags, such as -t.
		if e.Kind != KindCommand && len(e.Old) > 1 {
			assert.Contains(t, strings.ToLower(e.Old), "team", "old name should be team-related")
		}
	}
}

// TestEntriesCoverDeprecatedSources fails when a source that accepts a team-related name as a deprecated alias has one that is missing
// from the table, so no old-to-new pair is lost when those sources are removed. Each subtest scans one source and is deleted by the
// change that removes that source: it fails, rather than passing vacuously, once there is nothing left to scan.
func TestEntriesCoverDeprecatedSources(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	hasEntry := func(match func(Entry) bool) bool { return slices.ContainsFunc(teamEntries, match) }
	has := func(kind Kind, scope Scope, oldName, newName string) bool {
		return hasEntry(func(e Entry) bool {
			return e.Kind == kind && e.Scope&scope != 0 && e.Old == oldName && (newName == "" || e.New == newName)
		})
	}
	isTeamRelated := func(name string) bool { return strings.Contains(strings.ToLower(name), "team") }

	t.Run("renameto struct tags", func(t *testing.T) {
		t.Parallel()
		found := 0
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
				for key, kind := range map[string]Kind{
					"json": KindJSONKey, "query": KindQueryParam, "url": KindQueryParam, "form": KindFormField, "csv": KindCSVColumn,
				} {
					oldName, _, _ := strings.Cut(tag.Get(key), ",")
					if renameTo == "" || !isTeamRelated(oldName) {
						continue
					}
					found++
					scope := ScopeRequest | ScopeResponse
					if filepath.ToSlash(rel) == "server/fleet/activities.go" {
						scope = ScopeActivityDetails
					}
					assert.True(t, has(kind, scope, oldName, renameTo), "%s: %s tag %q renamed to %q is missing from entries", rel, key, oldName, renameTo)
				}
				return true
			})
		}
		assert.Positive(t, found)
	})

	t.Run("deprecated team paths", func(t *testing.T) {
		t.Parallel()
		teamPath := regexp.MustCompile(`/teams?(/|$)`)
		paramRegex := regexp.MustCompile(`\{(\w+):[^}]*\}`)
		found := 0
		ast.Inspect(parseFile(t, filepath.Join(root, "server", "service", "handler_deprecated_paths.go")), func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			primary := literalsOf(lit, "PrimaryPath")
			for _, deprecated := range literalsOf(lit, "DeprecatedPaths") {
				if len(primary) != 1 || !teamPath.MatchString(deprecated) {
					continue
				}
				found++
				oldPath, newPath := paramRegex.ReplaceAllString(deprecated, "{$1}"), paramRegex.ReplaceAllString(primary[0], "{$1}")
				assert.True(t, hasEntry(func(e Entry) bool { return e.Kind == KindPath && e.Old == oldPath && e.New == newPath }),
					"deprecated path %s -> %s is missing from entries", oldPath, newPath)
			}
			return true
		})
		assert.Positive(t, found)
	})

	t.Run("fleetctl flag, command, and env var aliases", func(t *testing.T) {
		t.Parallel()
		endsWithAny := func(context string, names []string) bool {
			return slices.ContainsFunc(names, func(name string) bool { return strings.HasSuffix(context, " "+name) })
		}
		found := 0
		for rel, file := range parseFiles(t, root, []string{filepath.Join("cmd", "fleetctl", "fleetctl")}, "") {
			for _, alias := range collectCLIAliases(file, isTeamRelated) {
				found++
				var match func(Entry) bool
				switch alias.kind {
				case cliCommandAlias:
					match = func(e Entry) bool {
						return e.Kind == KindCommand && strings.HasSuffix(e.Old, " "+alias.name) && strings.HasSuffix(e.New, " "+alias.command[0])
					}
				case cliFlagAlias:
					match = func(e Entry) bool {
						return e.Kind == KindFlag && e.Old == alias.name && endsWithAny(e.Context, alias.command) &&
							(alias.flagName == "" || e.New == alias.flagName)
					}
				case cliEnvVar:
					match = func(e Entry) bool {
						return e.Kind == KindEnvVar && e.Old == alias.name && endsWithAny(e.Context, alias.command)
					}
				}
				assert.True(t, hasEntry(match), "%s: %s %q of command %v is missing from entries", rel, alias.kind, alias.name, alias.command)
			}
		}
		assert.Positive(t, found)
	})

	t.Run("gitops key mappings", func(t *testing.T) {
		t.Parallel()
		found := 0
		ast.Inspect(parseFile(t, filepath.Join(root, "pkg", "spec", "gitops_deprecations.go")), func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if pair := stringLits(lit); len(lit.Elts) == 2 && len(pair) == 2 && isTeamRelated(pair[0]) {
				found++
				// A dotted path must have its own YAML entry; only an undotted name can be covered by a JSON key entry.
				matched := has(KindYAMLKey, ScopeRequest, pair[0], pair[1]) ||
					(!strings.Contains(pair[0], ".") && has(KindJSONKey, ScopeRequest, pair[0], pair[1]))
				assert.True(t, matched, "gitops mapping %s -> %s is missing from entries", pair[0], pair[1])
			}
			return true
		})
		assert.Positive(t, found)
	})

	t.Run("gitops no-team file and name", func(t *testing.T) {
		t.Parallel()
		var fileNames, noTeamNames []string
		ast.Inspect(parseFile(t, filepath.Join(root, "pkg", "spec", "gitops.go")), func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.BasicLit:
				if strings.HasSuffix(v.Value, `-team.yml"`) {
					fileNames = append(fileNames, stringLits(v)...)
				}
			case *ast.ValueSpec:
				if len(v.Names) == 1 && v.Names[0].Name == "noTeam" && len(v.Values) == 1 {
					noTeamNames = append(noTeamNames, stringLits(v.Values[0])...)
				}
			}
			return true
		})
		require.NotEmpty(t, fileNames)
		require.NotEmpty(t, noTeamNames)
		for _, name := range fileNames {
			assert.True(t, has(KindFileName, ScopeRequest, name, ""), "GitOps file name %s is missing from entries", name)
		}
		for _, name := range noTeamNames {
			assert.True(t, has(KindYAMLValue, ScopeRequest, "name: "+name, ""), "GitOps fleet name %s is missing from entries", name)
		}
	})
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

// stringLits returns the string literals of e, which is either one literal or a composite literal of them.
func stringLits(e ast.Expr) []string {
	var exprs []ast.Expr
	switch v := e.(type) {
	case *ast.CompositeLit:
		exprs = v.Elts
	default:
		exprs = []ast.Expr{e}
	}
	var out []string
	for _, expr := range exprs {
		if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, s)
			}
		}
	}
	return out
}

// literalsOf returns the string literals assigned to the given key of a composite literal.
func literalsOf(lit *ast.CompositeLit, key string) []string {
	var out []string
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == key {
				out = append(out, stringLits(kv.Value)...)
			}
		}
	}
	return out
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

// collectCLIAliases returns the command aliases, flag aliases, and env vars in file whose name satisfies match, with the command that
// declares each.
func collectCLIAliases(file *ast.File, match func(string) bool) []cliAlias {
	var out []cliAlias
	add := func(kind string, names, command []string, flagName string) {
		for _, name := range names {
			if match(name) {
				out = append(out, cliAlias{kind: kind, name: name, command: command, flagName: flagName})
			}
		}
	}
	var walk func(node ast.Node, command []string)
	walk = func(node ast.Node, command []string) {
		ast.Inspect(node, func(n ast.Node) bool {
			// logDeprecatedCommandName lists every deprecated spelling of a command, including ones with no team in their name.
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "logDeprecatedCommandName" && len(call.Args) == 3 {
					names, replacement := stringLits(call.Args[1]), stringLits(call.Args[2])
					if len(replacement) == 1 && slices.ContainsFunc(names, match) {
						for _, name := range names {
							out = append(out, cliAlias{kind: cliCommandAlias, name: name, command: replacement})
						}
					}
				}
				return true
			}
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Command" {
				if names := slices.Concat(literalsOf(lit, "Name"), literalsOf(lit, "Aliases")); len(names) > 0 {
					add(cliCommandAlias, literalsOf(lit, "Aliases"), names, "")
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
			// A flag is deprecated as a whole when one of its aliases names a team, so its other aliases, such as -t, are too.
			if aliases := literalsOf(lit, "Aliases"); slices.ContainsFunc(aliases, match) {
				for _, alias := range aliases {
					out = append(out, cliAlias{kind: cliFlagAlias, name: alias, command: command, flagName: flagName})
				}
			}
			add(cliEnvVar, literalsOf(lit, "EnvVars"), command, "")
			return true
		})
	}
	walk(file, nil)
	return out
}
