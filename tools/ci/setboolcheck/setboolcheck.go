// Package setboolcheck defines an analyzer that flags map[T]bool variables
// used as sets, suggesting map[T]struct{} instead.
//
// The heuristic: a local variable of type map[T]bool is flagged when every
// indexed assignment in the enclosing package uses the literal true as the
// value. This avoids false positives on maps that genuinely store bool values
// (e.g. map[string]bool{"a": true, "b": false}).
package setboolcheck

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer checks for map[T]bool that should be map[T]struct{}.
var Analyzer = &analysis.Analyzer{
	Name:     "setboolcheck",
	Doc:      "checks for map[T]bool that should be map[T]struct{} (set pattern)",
	URL:      "https://github.com/fleetdm/fleet",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// varInfo tracks analysis state for a single map[T]bool variable.
type varInfo struct {
	pos      token.Pos // declaration position (for diagnostics)
	hasAssign bool     // at least one indexed assignment or composite literal element
	allTrue  bool      // every assigned value is the literal true
	tainted  bool      // variable was assigned from an unknown source
	skip     bool      // function parameter or exported package-level var
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	vars := make(map[*types.Var]*varInfo)

	// Phase 1: collect map[T]bool variable declarations.
	declFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.GenDecl)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.RangeStmt)(nil),
	}

	insp.Preorder(declFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			registerParams(pass, vars, node)
		case *ast.GenDecl:
			if node.Tok == token.VAR {
				registerVarDecl(pass, vars, node)
			}
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				registerShortVarDecl(pass, vars, node)
			}
		case *ast.RangeStmt:
			if node.Tok == token.DEFINE {
				registerRangeVarDecl(pass, vars, node)
			}
		}
	})

	// Phase 2: scan all assignments for indexed writes and full reassignments.
	assignFilter := []ast.Node{(*ast.AssignStmt)(nil)}

	insp.Preorder(assignFilter, func(n ast.Node) {
		checkAssignment(pass, vars, n.(*ast.AssignStmt))
	})

	// Phase 3: report.
	for obj, info := range vars {
		if info.skip || info.tainted {
			continue
		}
		if info.hasAssign && info.allTrue {
			keyStr := obj.Type().Underlying().(*types.Map).Key().String()
			pass.Reportf(info.pos, "map[%s]bool used as a set; consider map[%s]struct{} instead", keyStr, keyStr)
		}
	}

	return nil, nil
}

// registerParams marks function parameters so they are skipped.
func registerParams(pass *analysis.Pass, vars map[*types.Var]*varInfo, fn *ast.FuncDecl) {
	if fn.Type.Params == nil {
		return
	}
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			if obj := asVar(pass, name); obj != nil && isBoolMap(obj.Type()) {
				vars[obj] = &varInfo{pos: name.Pos(), allTrue: true, skip: true}
			}
		}
	}
}

// registerVarDecl handles "var x map[T]bool" and "var x = make(map[T]bool)".
func registerVarDecl(pass *analysis.Pass, vars map[*types.Var]*varInfo, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			obj := asVar(pass, name)
			if obj == nil || !isBoolMap(obj.Type()) {
				continue
			}
			info := &varInfo{pos: name.Pos(), allTrue: true}

			if obj.Parent() == pass.Pkg.Scope() && obj.Exported() {
				info.skip = true
			}

			if i < len(vs.Values) {
				classifyInit(info, vs.Values[i])
			}
			vars[obj] = info
		}
	}
}

// registerShortVarDecl handles "x := make(map[T]bool)" and similar.
func registerShortVarDecl(pass *analysis.Pass, vars map[*types.Var]*varInfo, stmt *ast.AssignStmt) {
	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		obj := asVar(pass, ident)
		if obj == nil || !isBoolMap(obj.Type()) {
			continue
		}
		info := &varInfo{pos: ident.Pos(), allTrue: true}
		if i < len(stmt.Rhs) {
			classifyInit(info, stmt.Rhs[i])
		} else {
			info.tainted = true // multi-return
		}
		vars[obj] = info
	}
}

// registerRangeVarDecl marks range loop variables as tainted (unknown source).
func registerRangeVarDecl(pass *analysis.Pass, vars map[*types.Var]*varInfo, stmt *ast.RangeStmt) {
	for _, expr := range []ast.Expr{stmt.Key, stmt.Value} {
		if expr == nil {
			continue
		}
		ident, ok := expr.(*ast.Ident)
		if !ok {
			continue
		}
		if obj := asVar(pass, ident); obj != nil && isBoolMap(obj.Type()) {
			vars[obj] = &varInfo{pos: ident.Pos(), allTrue: true, tainted: true}
		}
	}
}

// classifyInit determines whether an initializer is safe (make, composite literal)
// or unknown (function call, variable copy, etc.).
func classifyInit(info *varInfo, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok && ident.Name == "make" {
			return // make(map[T]bool) — safe, no values yet
		}
		info.tainted = true // some other function call

	case *ast.CompositeLit:
		checkCompositeLitValues(info, e)

	default:
		info.tainted = true // variable copy, field access, etc.
	}
}

// checkCompositeLitValues checks that all values in a map literal are the literal true.
func checkCompositeLitValues(info *varInfo, lit *ast.CompositeLit) {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			info.allTrue = false
			break
		}
		info.hasAssign = true
		if !isTrueLiteral(kv.Value) {
			info.allTrue = false
			break
		}
	}
}

// checkAssignment examines assignments for indexed map writes and full reassignments.
func checkAssignment(pass *analysis.Pass, vars map[*types.Var]*varInfo, stmt *ast.AssignStmt) {
	// Check indexed assignments: m[k] = value
	for i, lhs := range stmt.Lhs {
		indexExpr, ok := lhs.(*ast.IndexExpr)
		if !ok {
			continue
		}
		obj := identVar(pass, indexExpr.X)
		if obj == nil {
			continue
		}
		info, ok := vars[obj]
		if !ok {
			continue
		}
		info.hasAssign = true
		if rhs := correspondingRHS(stmt, i); rhs == nil || !isTrueLiteral(rhs) {
			info.allTrue = false
		}
	}

	// Detect full reassignment of a tracked variable: m = ...
	if stmt.Tok != token.ASSIGN {
		return
	}
	for i, lhs := range stmt.Lhs {
		// Skip index expressions — already handled above.
		if _, isIndex := lhs.(*ast.IndexExpr); isIndex {
			continue
		}
		obj := identVar(pass, lhs)
		if obj == nil {
			continue
		}
		info, ok := vars[obj]
		if !ok {
			continue
		}
		rhs := correspondingRHS(stmt, i)
		if rhs == nil {
			info.tainted = true
			continue
		}
		switch e := rhs.(type) {
		case *ast.CallExpr:
			if id, ok2 := e.Fun.(*ast.Ident); ok2 && id.Name == "make" {
				continue // re-init with make — safe
			}
			info.tainted = true
		case *ast.CompositeLit:
			checkCompositeLitValues(info, e)
		default:
			info.tainted = true
		}
	}
}

// --- helpers ---

// asVar resolves an ast.Ident to a *types.Var, or returns nil.
func asVar(pass *analysis.Pass, ident *ast.Ident) *types.Var {
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil
	}
	v, ok := obj.(*types.Var)
	if !ok {
		return nil
	}
	return v
}

// identVar extracts the *types.Var from an expression if it is a simple identifier.
func identVar(pass *analysis.Pass, expr ast.Expr) *types.Var {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}
	return asVar(pass, ident)
}

// isBoolMap reports whether t is (or has underlying type) map[T]bool.
func isBoolMap(t types.Type) bool {
	m, ok := t.Underlying().(*types.Map)
	if !ok {
		return false
	}
	b, ok := m.Elem().(*types.Basic)
	return ok && b.Kind() == types.Bool
}

// isTrueLiteral reports whether expr is the unqualified identifier "true".
func isTrueLiteral(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "true"
}

// correspondingRHS returns the RHS expression matching LHS index i,
// or nil for multi-value returns.
func correspondingRHS(stmt *ast.AssignStmt, i int) ast.Expr {
	if len(stmt.Lhs) == len(stmt.Rhs) {
		return stmt.Rhs[i]
	}
	return nil
}
