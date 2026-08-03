package main

import (
	"strconv"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer runs the analyzer against testdata with a deliberately tiny limit, so the fixture does not need a genuinely
// 500-block function to exercise the reporting path.
func TestAnalyzer(t *testing.T) {
	const testMax = 5

	if err := analyzer.Flags.Set("max", strconv.Itoa(testMax)); err != nil {
		t.Fatalf("set max flag: %s", err)
	}
	t.Cleanup(func() {
		if err := analyzer.Flags.Set("max", strconv.Itoa(defaultMaxCFGBlocks)); err != nil {
			t.Fatalf("restore max flag: %s", err)
		}
	})

	analysistest.Run(t, analysistest.TestData(), analyzer, "example")
}

// TestDefaultMatchesNilaway guards the constant against drifting above nilaway's own limit, where the gate would stop meaning
// anything.
func TestDefaultMatchesNilaway(t *testing.T) {
	const nilawayMaxFuncSizeInCFGBlocks = 500

	if defaultMaxCFGBlocks > nilawayMaxFuncSizeInCFGBlocks {
		t.Errorf("defaultMaxCFGBlocks = %d, must not exceed nilaway's limit of %d",
			defaultMaxCFGBlocks, nilawayMaxFuncSizeInCFGBlocks)
	}
}
