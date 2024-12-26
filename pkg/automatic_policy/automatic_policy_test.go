package automatic_policy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateErrors(t *testing.T) {
	_, err := Generate(InstallerMetadata{
		Title:            "Foobar",
		Extension:        "exe",
		BundleIdentifier: "",
		PackageIDs:       []string{"Foobar"},
	})
	require.ErrorIs(t, err, ErrExtensionNotSupported)

	_, err = Generate(InstallerMetadata{
		Title:            "Foobar",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingProductCode)
	_, err = Generate(InstallerMetadata{
		Title:            "Foobar",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{},
	})
	require.ErrorIs(t, err, ErrMissingProductCode)

	_, err = Generate(InstallerMetadata{
		Title:            "Foobar",
		Extension:        "pkg",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingBundleIdentifier)

	_, err = Generate(InstallerMetadata{
		Title:            "",
		Extension:        "deb",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingTitle)
}

func TestGenerate(t *testing.T) {
	policyData, err := Generate(InstallerMetadata{
		Title:            "Foobar",
		Extension:        "pkg",
		BundleIdentifier: "com.foo.bar",
		PackageIDs:       []string{"com.foo.bar"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Foobar (pkg)", policyData.Name)
	require.Equal(t, "Policy triggers automatic install of Foobar on each host that's missing this software.", policyData.Description)
	require.Equal(t, "darwin", policyData.Platform)
	require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo.bar';", policyData.Query)

	policyData, err = Generate(InstallerMetadata{
		Title:            "Barfoo",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{"foo"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Barfoo (msi)", policyData.Name)
	require.Equal(t, "Policy triggers automatic install of Barfoo on each host that's missing this software.", policyData.Description)
	require.Equal(t, "windows", policyData.Platform)
	require.Equal(t, "SELECT 1 FROM programs WHERE identifying_number = 'foo';", policyData.Query)

	policyData, err = Generate(InstallerMetadata{
		Title:            "Zoobar",
		Extension:        "deb",
		BundleIdentifier: "",
		PackageIDs:       []string{"Zoobar"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Zoobar (deb)", policyData.Name)
	require.Equal(t, `Policy triggers automatic install of Zoobar on each host that's missing this software.
Software won't be installed on Linux hosts with Debian-based distributions because this policy's query is written to always pass on these hosts.`, policyData.Description)
	require.Equal(t, "linux", policyData.Platform)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 FROM os_version WHERE platform = 'rhel'
) OR EXISTS (
	SELECT 1 FROM deb_packages WHERE name = 'Zoobar'
);`, policyData.Query)

	policyData, err = Generate(InstallerMetadata{
		Title:            "Barzoo",
		Extension:        "rpm",
		BundleIdentifier: "",
		PackageIDs:       []string{"Barzoo"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Barzoo (rpm)", policyData.Name)
	require.Equal(t, `Policy triggers automatic install of Barzoo on each host that's missing this software.
Software won't be installed on Linux hosts with RPM-based distributions because this policy's query is written to always pass on these hosts.`, policyData.Description)
	require.Equal(t, "linux", policyData.Platform)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 FROM os_version WHERE platform != 'rhel'
) OR EXISTS (
	SELECT 1 FROM rpm_packages WHERE name = 'Barzoo'
);`, policyData.Query)
}
