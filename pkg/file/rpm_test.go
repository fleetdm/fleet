package file

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/goreleaser/nfpm/v2/rpm"
	"github.com/stretchr/testify/require"
)

func TestExtractRPMMetadata(t *testing.T) {
	//
	// Build an RPM package on the fly with nfpm.
	//
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "foo.sh"), []byte("#!/bin/sh\n\necho \"Foo!\"\n"), constant.DefaultFileMode)
	require.NoError(t, err)
	contents := files.Contents{
		&files.Content{
			Source:      filepath.Join(tmpDir, "**"),
			Destination: "/",
		},
	}
	postInstallPath := filepath.Join(t.TempDir(), "postinstall.sh")
	err = os.WriteFile(postInstallPath, []byte("#!/bin/sh\n\necho \"Hello world!\"\n"), constant.DefaultFileMode)
	require.NoError(t, err)
	info := &nfpm.Info{
		Name:        "foobar",
		Version:     "1.2.3",
		Description: "Foo bar",
		Arch:        "x86_64",
		Maintainer:  "Fleet Device Management",
		Vendor:      "Fleet Device Management",
		License:     "LICENSE",
		Homepage:    "https://example.com",
		Overridables: nfpm.Overridables{
			Contents: contents,
			Scripts: nfpm.Scripts{
				PostInstall: postInstallPath,
			},
		},
	}
	rpmPath := filepath.Join(t.TempDir(), "foobar.rpm")
	out, err := os.OpenFile(rpmPath, os.O_CREATE|os.O_RDWR, constant.DefaultFileMode)
	require.NoError(t, err)
	t.Cleanup(func() {
		out.Close()
	})
	err = rpm.Default.Package(info, out)
	require.NoError(t, err)
	err = out.Close()
	require.NoError(t, err)

	//
	// Test ExtractRPMMetadata with the generated package.
	// Using ExtractInstallerMetadata for broader testing (for a file
	// with rpm extension it will call ExtractRPMMetadata).
	//
	tfr, err := fleet.NewKeepFileReader(rpmPath)
	require.NoError(t, err)
	t.Cleanup(func() { tfr.Close() })
	m, err := ExtractInstallerMetadata(tfr)
	require.NoError(t, err)
	require.Empty(t, m.BundleIdentifier)
	require.Equal(t, "rpm", m.Extension)
	require.Equal(t, "foobar", m.Name)
	require.Equal(t, []string{"foobar"}, m.PackageIDs)
	require.Equal(t, sha256FilePath(t, rpmPath), m.SHASum)
	require.Equal(t, "1.2.3", m.Version)
}

func sha256FilePath(t *testing.T, path string) []byte {
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		f.Close()
	})
	h := sha256.New()
	_, err = io.Copy(h, f)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	return h.Sum(nil)
}
