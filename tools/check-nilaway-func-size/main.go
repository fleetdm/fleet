// Command check-nilaway-func-size fails the build if any Go function is too large for nilaway to analyze.
//
// Background: nilaway skips any function whose control-flow graph exceeds a fixed block count (_maxFuncSizeInCFGBlocks, currently
// 500, in go.uber.org/nilaway/assertion/function/analyzer.go). A skipped function is not merely unanalyzed on its own. nilaway's
// accumulation analyzer bails out for the whole package as soon as the assertion analyzer reports any error, so a single
// oversized function costs every other function in that package its nil-panic analysis, and costs dependent packages the
// inference facts that package would have exported.
//
// This tool consumes the same golang.org/x/tools/go/analysis/passes/ctrlflow CFGs that nilaway consumes, so its block counts are
// identical to nilaway's by construction rather than an approximation of them.
//
// Usage:
//
//	go run ./tools/check-nilaway-func-size ./...
//
// Wired into make lint-go (see Makefile).
package main

import (
	"fmt"
	"go/ast"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// defaultMaxCFGBlocks mirrors nilaway's own limit. Lowering it via -max buys headroom so functions are flagged before they
// actually lose analysis. Raising it above the limit is rejected.
const defaultMaxCFGBlocks = 500

var maxCFGBlocks = defaultMaxCFGBlocks

var analyzer = &analysis.Analyzer{
	Name:     "nilawayfuncsize",
	Doc:      "reports functions with too many CFG blocks for nilaway to analyze",
	Requires: []*analysis.Analyzer{ctrlflow.Analyzer},
	Run:      run,
}

func init() {
	analyzer.Flags.Func("max",
		fmt.Sprintf("maximum CFG blocks allowed per function, between 1 and nilaway's own limit of %d (default %d)",
			defaultMaxCFGBlocks, defaultMaxCFGBlocks),
		func(value string) error {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("must be an integer, got %q", value)
			}
			if n < 1 {
				return fmt.Errorf("must be at least 1, got %d", n)
			}
			if n > defaultMaxCFGBlocks {
				return fmt.Errorf("must not exceed nilaway's own limit of %d, got %d", defaultMaxCFGBlocks, n)
			}
			maxCFGBlocks = n
			return nil
		})
}

func run(pass *analysis.Pass) (any, error) {
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// ctrlflow.CFGs.FuncDecl looks the declaration up by its type object and would panic on a name that has none, as in a blank "func
			// _()" declaration. nilaway never sizes those either, so skip them.
			if pass.TypesInfo.Defs[fn.Name] == nil {
				continue
			}
			// Only function declarations are gated. nilaway size-checks function literals solely when its experimental-anonymous-function
			// flag is set, and .golangci-incremental.yml does not set it, so an oversized closure costs us nothing today.
			graph := cfgs.FuncDecl(fn)
			if graph == nil {
				// ctrlflow builds no CFG for functions in its hard-coded known-intrinsic list (log.Fatal and friends). nilaway skips those as well.
				continue
			}
			if len(graph.Blocks) > maxCFGBlocks {
				pass.Reportf(fn.Pos(),
					"%s has %d CFG blocks, over the limit of %d. nilaway skips functions this large, and drops "+
						"nil-panic analysis for every other function in the package with it. Split it into helpers. ",
					fn.Name.Name, len(graph.Blocks), maxCFGBlocks)
			}
		}
	}

	return nil, nil
}

func main() { singlechecker.Main(analyzer) }
