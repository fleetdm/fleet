// Package apiparamcheck defines an analyzer that flags json/url/query struct
// tags whose name contains deprecated Fleet terminology.
//
// Two renames are enforced:
//   - "team" / "teams" was renamed to "fleet" / "fleets". Any occurrence of
//     "team" or "teams" as a snake_case token in a tag name is flagged.
//   - "query" / "queries" was renamed to "report" / "reports" when referring
//     to the product concept (the SQL sense of the word is fine). A tag name
//     of exactly "query" or "queries" is allowed; any larger name containing
//     "query" or "queries" as a snake_case token is flagged.
package apiparamcheck

import (
	"go/ast"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer flags struct tags using deprecated "team"/"teams"/"query"/"queries"
// terminology in json/url/query tag names.
var Analyzer = &analysis.Analyzer{
	Name:     "apiparamcheck",
	Doc:      "flags json/url/query struct tags using deprecated team/teams or query/queries terms",
	URL:      "https://github.com/fleetdm/fleet",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var checkedTagKeys = []string{"json", "url", "query"}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.StructType)(nil)}, func(n ast.Node) {
		st := n.(*ast.StructType)
		if st.Fields == nil {
			return
		}
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			raw := field.Tag.Value
			if len(raw) < 2 {
				continue
			}
			tag := reflect.StructTag(strings.Trim(raw, "`"))
			for _, key := range checkedTagKeys {
				v, ok := tag.Lookup(key)
				if !ok {
					continue
				}
				name, _, _ := strings.Cut(v, ",")
				if msg := violationMessage(name); msg != "" {
					pass.Reportf(field.Tag.Pos(), "%s tag %q: %s", key, name, msg)
				}
			}
		}
	})

	return nil, nil
}

// violationMessage returns a non-empty message describing why the tag name
// is invalid, or "" if the name is allowed.
func violationMessage(name string) string {
	tokens := splitTokens(name)
	if len(tokens) == 0 {
		return ""
	}
	for _, t := range tokens {
		low := strings.ToLower(t)
		if low == "team" || low == "teams" {
			return `uses deprecated "team"/"teams" — use "fleet"/"fleets" instead`
		}
	}
	// "query"/"queries" are allowed when they are the entire tag name
	// (referring to a SQL query), but not as part of a larger name.
	if len(tokens) > 1 {
		for _, t := range tokens {
			low := strings.ToLower(t)
			if low == "query" || low == "queries" {
				return `uses "query"/"queries" as part of a name — use "report"/"reports" instead (bare "query"/"queries" is ok)`
			}
		}
	}
	return ""
}

// splitTokens splits a tag name into its snake_case and camelCase components.
// Examples:
//
//	"team_id"          -> ["team", "id"]
//	"hostTeamID"       -> ["host", "Team", "ID"]
//	"query"            -> ["query"]
//	"osquery_version"  -> ["osquery", "version"]
func splitTokens(name string) []string {
	if name == "" {
		return nil
	}
	// First split on underscores and hyphens (snake_case / kebab-case).
	var out []string
	for _, part := range strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		out = append(out, splitCamel(part)...)
	}
	return out
}

// splitCamel splits a camelCase or PascalCase identifier into its components.
// Runs of consecutive uppercase letters are kept together (e.g. "ID").
func splitCamel(s string) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	var parts []string
	start := 0
	for i := 1; i < len(runes); i++ {
		cur, prev := runes[i], runes[i-1]
		// Boundary: lowercase/digit followed by uppercase.
		if isUpper(cur) && !isUpper(prev) {
			parts = append(parts, string(runes[start:i]))
			start = i
			continue
		}
		// Boundary: end of an uppercase run before a lowercase
		// (e.g. "IDs" -> "I", "Ds" is wrong; "IDs" should split as
		// "ID"+"s" — detect the split one rune earlier).
		if i+1 < len(runes) && isUpper(prev) && isUpper(cur) && !isUpper(runes[i+1]) {
			parts = append(parts, string(runes[start:i]))
			start = i
			continue
		}
	}
	parts = append(parts, string(runes[start:]))
	return parts
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
