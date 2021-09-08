package main

import (
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestPackage(t *testing.T) {

	// --type is required
	runAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -fleet-url & --enroll-secret are required together
	runAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"}, "--enroll-secret and --fleet-url must be provided together")
	runAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")

	// --insecure and --fleet-certificate are mutually exclusive
	runAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"}, "--insecure and --fleet-certificate may not be provided together")

	// run package tests, each should output their respective package type
	// orbit-osquery_0.0.3_amd64.deb
	runAppForTest(t, []string{"package", "--type=deb", "--insecure"})
	// orbit-osquery-0.0.3.x86_64.rpm
	runAppForTest(t, []string{"package", "--type=rpm", "--insecure"})
	// orbit-osquery_0.0.3.msi
	//runAppForTest(t, []string{"package", "--type=msi", "--insecure"}) TODO: this is currently failing on Github runners due to permission issues

	//runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu

	dir, err := os.Getwd()
	require.NoError(t, err)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		// TODO validate contents
		switch filepath.Ext(path) {
		case ".msi", ".deb", ".rpm", ".pkg":
			info, err := d.Info()
			require.NoError(t, err)
			require.Greater(t, info.Size(), int64(0))
		}
		return nil
	})

}
