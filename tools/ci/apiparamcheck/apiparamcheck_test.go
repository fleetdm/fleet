package apiparamcheck_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/tools/ci/apiparamcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, apiparamcheck.Analyzer, "example")
}
