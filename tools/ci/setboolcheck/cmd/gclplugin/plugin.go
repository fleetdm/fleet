// Package gclplugin provides the golangci-lint module plugin entry point.
package gclplugin

import (
	"github.com/fleetdm/fleet/v4/tools/ci/setboolcheck"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("setboolcheck", New)
}

// New returns the golangci-lint plugin for the setboolcheck analyzer.
func New(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{setboolcheck.Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
