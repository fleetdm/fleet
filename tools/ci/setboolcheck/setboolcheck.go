// Package setboolcheck defines an analyzer that flags map[T]bool variables
// used as sets, suggesting map[T]struct{} instead.
//
// The heuristic: a map[T]bool variable (local or unexported package-level)
// is flagged when, in the enclosing package, every observed write via
// indexed assignments or composite literal elements uses the literal true
// as the value. This avoids false positives on maps that genuinely store
// bool values (e.g. map[string]bool{"a": true, "b": false}).
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
	pos       token.Pos // declaration position (for diagnostics)
	hasAssign bool      // at least one indexed assignment or composite literal element
	allTrue   bool      // every assigned value is the literal true
	tainted   bool      // variable was assigned from an unknown source
	skip      bool      // function parameter or exported package-level var
}

func run(pass *analysis.Pass) (any, error) {
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

	// Phase 2: scan assignments for indexed writes, full reassignments, and escapes.
	escapeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(escapeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.AssignStmt:
			checkAssignment(pass, vars, node)
			// Taint tracked maps that appear on the RHS of assignments (aliasing).
			taintEscapedInAssign(pass, vars, node)
		case *ast.CallExpr:
			// Taint tracked maps passed as function arguments.
			taintEscapedInCall(pass, vars, node)
		}
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

			if len(vs.Names) == len(vs.Values) && i < len(vs.Values) {
				classifyInit(pass, info, vs.Values[i])
			} else if len(vs.Values) > 0 {
				info.tainted = true // multi-return or mismatched initializers
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
			classifyInit(pass, info, stmt.Rhs[i])
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
func classifyInit(pass *analysis.Pass, info *varInfo, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if isBuiltinMake(pass, e) {
			return // make(map[T]bool) - safe, no values yet
		}
		info.tainted = true // some other function call

	case *ast.CompositeLit:
		checkCompositeLitValues(pass, info, e)

	default:
		info.tainted = true // variable copy, field access, etc.
	}
}

// checkCompositeLitValues checks that all values in a map literal are the literal true.
func checkCompositeLitValues(pass *analysis.Pass, info *varInfo, lit *ast.CompositeLit) {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			info.allTrue = false
			break
		}
		info.hasAssign = true
		if !isBuiltinTrue(pass, kv.Value) {
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
		if rhs := correspondingRHS(stmt, i); rhs == nil || !isBuiltinTrue(pass, rhs) {
			info.allTrue = false
		}
	}

	// Detect full reassignment of a tracked variable: m = ...
	if stmt.Tok != token.ASSIGN {
		return
	}
	for i, lhs := range stmt.Lhs {
		// Skip index expressions - already handled above.
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
			if isBuiltinMake(pass, e) {
				continue // re-init with make - safe
			}
			info.tainted = true
		case *ast.CompositeLit:
			checkCompositeLitValues(pass, info, e)
		default:
			info.tainted = true
		}
	}
}

// taintEscapedInAssign taints tracked maps that appear on the RHS of an assignment
// to another named variable, since the alias could be used to write non-true values.
// Assignments to the blank identifier (_ = m) are ignored since they are no-ops.
func taintEscapedInAssign(pass *analysis.Pass, vars map[*types.Var]*varInfo, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		obj := identVar(pass, rhs)
		if obj == nil {
			continue
		}
		if _, ok := vars[obj]; !ok {
			continue
		}
		// Only taint if the LHS is a real named variable (not blank identifier).
		if i < len(stmt.Lhs) {
			if lhsIdent, ok := stmt.Lhs[i].(*ast.Ident); ok && lhsIdent.Name == "_" {
				continue
			}
		}
		vars[obj].tainted = true
	}
}

// taintEscapedInCall taints tracked maps passed as function arguments,
// since the callee could write non-true values through the reference.
func taintEscapedInCall(pass *analysis.Pass, vars map[*types.Var]*varInfo, call *ast.CallExpr) {
	// Skip the builtin make - its arguments are types, not map values.
	if isBuiltinMake(pass, call) {
		return
	}
	for _, arg := range call.Args {
		obj := identVar(pass, arg)
		if obj == nil {
			continue
		}
		if info, ok := vars[obj]; ok {
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

// isBuiltinTrue reports whether expr is the predeclared identifier "true"
// (not a shadowed local variable named "true").
func isBuiltinTrue(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	return obj == types.Universe.Lookup("true")
}

// isBuiltinMake reports whether call is a call to the predeclared "make" function.
func isBuiltinMake(pass *analysis.Pass, call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	return obj == types.Universe.Lookup("make")
}

// correspondingRHS returns the RHS expression matching LHS index i,
// or nil for multi-value returns.
func correspondingRHS(stmt *ast.AssignStmt, i int) ast.Expr {
	if len(stmt.Lhs) == len(stmt.Rhs) {
		return stmt.Rhs[i]
	}
	return nil
}
