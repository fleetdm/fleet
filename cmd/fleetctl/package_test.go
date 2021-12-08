package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	// --type is required
	runAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -fleet-url & --enroll-secret are required together
	runAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"}, "--enroll-secret and --fleet-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")

	// --insecure and --fleet-certificate are mutually exclusive
	runAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"}, "--insecure and --fleet-certificate may not be provided together")

	// run package tests, each should output their respective package type
	// fleet-osquery_0.0.3_amd64.deb
	runAppForTest(t, []string{"package", "--type=deb", "--insecure"})
	info, err := os.Stat("fleet-osquery_0.0.3_amd64.deb")
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0)) // TODO verify contents
	// fleet-osquery-0.0.3.x86_64.rpm
	runAppForTest(t, []string{"package", "--type=rpm", "--insecure"})
	info, err = os.Stat("fleet-osquery-0.0.3.x86_64.rpm")
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0)) // TODO verify contents
	// fleet-osquery.msi
	// runAppForTest(t, []string{"package", "--type=msi", "--insecure"}) TODO: this is currently failing on Github runners due to permission issues
	// info, err = os.Stat("orbit-osquery_0.0.3.msi")
	// require.NoError(t, err)
	// require.Greater(t, info.Size(), int64(0))

	// runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu
}
