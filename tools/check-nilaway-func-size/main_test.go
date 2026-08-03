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

	analysistest.Run(t, analysistest.TestData(), analyzer, "oversized")
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

// TestMaxFlag checks that -max rejects values nilaway would ignore. A -max above nilaway's own limit would let this gate pass a
// function that nilaway still refuses to analyze, which defeats the point of the check.
func TestMaxFlag(t *testing.T) {
	t.Cleanup(func() {
		if err := analyzer.Flags.Set("max", strconv.Itoa(defaultMaxCFGBlocks)); err != nil {
			t.Fatalf("restore max flag: %s", err)
		}
	})

	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "below the limit", value: "450", wantErr: false},
		{name: "exactly the limit", value: "500", wantErr: false},
		{name: "one over the limit", value: "501", wantErr: true},
		{name: "far over the limit", value: "10000", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "not an integer", value: "abc", wantErr: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.Flags.Set("max", tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("Set(max=%s) succeeded, want an error", tt.value)
			}
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Set(max=%s): %s", tt.value, err)
				}
				want, _ := strconv.Atoi(tt.value)
				if maxCFGBlocks != want {
					t.Errorf("maxCFGBlocks = %d, want %d", maxCFGBlocks, want)
				}
			}
		})
	}
}
