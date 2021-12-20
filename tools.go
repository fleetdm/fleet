//go:build tools
// +build tools

package tools

import (
	_ "github.com/fleetdm/goose"
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/kevinburke/go-bindata"
	_ "github.com/mitchellh/gon/cmd/gon"
)
