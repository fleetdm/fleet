// Package gclplugin provides the golangci-lint module plugin entry point.
package gclplugin

import (
	"github.com/fleetdm/fleet/v4/tools/ci/apiparamcheck"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("apiparamcheck", New)
}

// New returns the golangci-lint plugin for the apiparamcheck analyzer.
func New(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{apiparamcheck.Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
