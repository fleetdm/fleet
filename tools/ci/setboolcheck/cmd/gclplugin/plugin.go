// Package gclplugin provides the golangci-lint module plugin entry point.
package gclplugin

import (
	"github.com/fleetdm/fleet/v4/tools/ci/setboolcheck"
	"golang.org/x/tools/go/analysis"
)

// New is the entry point for the golangci-lint module plugin system.
func New(_ any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{setboolcheck.Analyzer}, nil
}
