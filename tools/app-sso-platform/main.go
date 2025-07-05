//go:build darwin
// +build darwin

//go:debug x509negativeserial=1

// Package main is a macOS application to test the app_sso_platform table in the command line.
// Usage for SSO Platform extension for Microsoft Company Portal:
// "go run ./tools/app-sso-platform com.microsoft.CompanyPortalMac.ssoextension KERBEROS.MICROSOFTONLINE.COM"
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/app_sso_platform"
	"github.com/osquery/osquery-go/plugin/table"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s <extensionIdentifier> <realm>\n", os.Args[0])
		os.Exit(1)
	}
	extensionIdentifier := os.Args[1]
	realm := os.Args[2]

	rows, err := app_sso_platform.Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"extension_identifier": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: extensionIdentifier,
					},
				},
			},
			"realm": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: realm,
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", rows)
}
