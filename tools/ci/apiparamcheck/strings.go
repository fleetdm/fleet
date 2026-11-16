package apiparamcheck

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// notFleetTeam matches uses of "team" that are not Fleet's renamed concept.
var notFleetTeam = regexp.MustCompile(`(?i)team[ _]?identifier|app_team_id|microsoft teams`)

// sqlText matches SQL statements and fragments, since table and column names (teams, team_id) keep the old name.
var sqlText = regexp.MustCompile(`\b(SELECT|INSERT INTO|UPDATE|DELETE FROM|FROM|JOIN|WHERE|ORDER BY|GROUP BY|IS NULL|IS NOT NULL|AND|OR)\b|` +
	`\b[a-z]{1,4}\.team_id\b|team_id\s*(=|IN\b|<=>)`)

// identifierOnly matches a literal with no spaces, such as a column name.
var identifierOnly = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)

// skippedPaths hold strings that use the old names on purpose.
var skippedPaths = []string{
	"/migrations/",
	"/pkg/removednames/",
	"/pkg/spec/gitops_deprecations.go",
	"/server/service/handler_removed_paths.go",
	"/tools/", // not shipped
}

// checkStrings reports string literals in non-test code that use "team" or "teams": messages, help text, log keys, and map keys.
func checkStrings(pass *analysis.Pass, insp *inspector.Inspector) {
	insp.WithStack([]ast.Node{(*ast.BasicLit)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		lit := n.(*ast.BasicLit)
		if !push || lit.Kind != token.STRING {
			return true
		}
		switch stack[len(stack)-2].(type) {
		case *ast.Field, *ast.ImportSpec: // struct tags are checked by tag key
			return true
		}
		filename := pass.Fset.Position(lit.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") || skipped(filename) {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil || isWrapMessage(stack, s) || sqlText.MatchString(s) ||
			(strings.Contains(filename, "/server/datastore/") && identifierOnly.MatchString(s)) {
			return true
		}
		for _, word := range strings.FieldsFunc(notFleetTeam.ReplaceAllString(s, " "), isNotAlnum) {
			for _, t := range splitCamel(word) {
				if low := strings.ToLower(t); low == "team" || low == "teams" {
					pass.Reportf(lit.Pos(), `string uses deprecated "team"/"teams": use "fleet"/"fleets" instead`)
					return true
				}
			}
		}
		return true
	})
}

func skipped(filename string) bool {
	if strings.Contains(filename, "/testdata/src/") { // this analyzer's own test data, under tools/
		return false
	}
	for _, p := range skippedPaths {
		if strings.Contains(filename, p) {
			return true
		}
	}
	return false
}

// isWrapMessage reports whether s describes a wrapped error. Wrap messages are logged, not shown to users.
func isWrapMessage(stack []ast.Node, s string) bool {
	pkg, fn := callee(stack[len(stack)-2])
	if pkg == "fmt" && fn == "Sprintf" && len(stack) > 2 { // ctxerr.Wrap(ctx, err, fmt.Sprintf(...))
		pkg, fn = callee(stack[len(stack)-3])
	}
	return (pkg == "ctxerr" && strings.HasPrefix(fn, "Wrap")) || (pkg == "fmt" && fn == "Errorf" && strings.Contains(s, "%w"))
}

// callee returns the package and function names of a call such as ctxerr.Wrap(...).
func callee(n ast.Node) (pkg, fn string) {
	if call, ok := n.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name, sel.Sel.Name
			}
		}
	}
	return "", ""
}

func isNotAlnum(r rune) bool {
	return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9')
}
