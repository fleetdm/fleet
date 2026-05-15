package setboolcheck_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/tools/ci/setboolcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, setboolcheck.Analyzer, "example")
}
