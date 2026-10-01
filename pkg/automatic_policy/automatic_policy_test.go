package automatic_policy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateErrors(t *testing.T) {
	_, err := Generate(FullInstallerMetadata{
		Title:            "Foobar",
		Extension:        "exe",
		BundleIdentifier: "",
		PackageIDs:       []string{"Foobar"},
	})
	require.ErrorIs(t, err, ErrExtensionNotSupported)

	_, err = FullInstallerMetadata{}.PolicyPlatform()
	require.ErrorIs(t, err, ErrExtensionNotSupported)

	_, err = Generate(FullInstallerMetadata{
		Title:            "Foobar",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingProductAndUpgradeCode)
	_, err = Generate(FullInstallerMetadata{
		Title:            "Foobar",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{},
	})
	require.ErrorIs(t, err, ErrMissingProductAndUpgradeCode)

	_, err = Generate(MacInstallerMetadata{
		Title:            "Foobar",
		BundleIdentifier: "",
	})
	require.ErrorIs(t, err, ErrMissingBundleIdentifier)

	_, err = Generate(FullInstallerMetadata{
		Title:            "Foobar",
		Extension:        "pkg",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingBundleIdentifier)

	_, err = Generate(MacInstallerMetadata{
		Title:            "",
		BundleIdentifier: "",
	})
	require.ErrorIs(t, err, ErrMissingTitle)

	_, err = MacInstallerMetadata{}.PolicyQuery()
	require.ErrorIs(t, err, ErrMissingBundleIdentifier)

	_, err = Generate(FullInstallerMetadata{
		Title:            "",
		Extension:        "deb",
		BundleIdentifier: "",
		PackageIDs:       []string{""},
	})
	require.ErrorIs(t, err, ErrMissingTitle)

	_, err = Generate(FMAInstallerMetadata{})
	require.ErrorIs(t, err, ErrMissingTitle)

	_, err = FMAInstallerMetadata{}.PolicyDescription()
	require.ErrorIs(t, err, ErrMissingTitle)
}

func TestGenerate(t *testing.T) {
	policyData, err := Generate(MacInstallerMetadata{
		Title:            "Foobar",
		BundleIdentifier: "com.foo.bar",
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Foobar", policyData.Name)
	require.Equal(t, "Policy triggers automatic install of Foobar on each host that's missing this software.", policyData.Description)
	require.Equal(t, "darwin", policyData.Platform)
	require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo.bar';", policyData.Query)

	policyData, err = Generate(FullInstallerMetadata{
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

	// MSI with only product code
	policyData, err = Generate(FullInstallerMetadata{
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

	// MSI with upgrade code
	policyData, err = Generate(FullInstallerMetadata{
		Title:            "Barfoo",
		Extension:        "msi",
		BundleIdentifier: "",
		PackageIDs:       []string{"foo"},
		UpgradeCode:      "bar",
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Barfoo (msi)", policyData.Name)
	require.Equal(t, "Policy triggers automatic install of Barfoo on each host that's missing this software.", policyData.Description)
	require.Equal(t, "windows", policyData.Platform)
	require.Equal(t, "SELECT 1 FROM programs WHERE upgrade_code = 'bar';", policyData.Query)

	policyData, err = Generate(FullInstallerMetadata{
		Title:            "Zoobar",
		Extension:        "deb",
		BundleIdentifier: "",
		PackageIDs:       []string{"Zoobar"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Zoobar (deb)", policyData.Name)
	require.Equal(t, `Policy triggers automatic install of Zoobar on each host that's missing this software.
Software won't be installed on Linux hosts with RPM-based distributions because this policy's query is written to always pass on these hosts.`, policyData.Description)
	require.Equal(t, "linux", policyData.Platform)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM deb_packages) = 0
) OR EXISTS (
	SELECT 1 FROM deb_packages WHERE name = 'Zoobar' AND status = 'install ok installed'
);`, policyData.Query)

	policyData, err = Generate(FullInstallerMetadata{
		Title:            "Barzoo",
		Extension:        "rpm",
		BundleIdentifier: "",
		PackageIDs:       []string{"Barzoo"},
	})
	require.NoError(t, err)
	require.Equal(t, "[Install software] Barzoo (rpm)", policyData.Name)
	require.Equal(t, `Policy triggers automatic install of Barzoo on each host that's missing this software.
Software won't be installed on Linux hosts with Debian-based distributions because this policy's query is written to always pass on these hosts.`, policyData.Description)
	require.Equal(t, "linux", policyData.Platform)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM rpm_packages) = 0
) OR EXISTS (
	SELECT 1 FROM rpm_packages WHERE name = 'Barzoo'
);`, policyData.Query)
}

func TestPolicyQueryEscapesQuotes(t *testing.T) {
	const payload = "x' UNION SELECT 1 FROM shadow --"
	const escaped = "x'' UNION SELECT 1 FROM shadow --"

	for _, tc := range []struct {
		name string
		meta InstallerMetadata
		want string
	}{
		{"mac bundle identifier", MacInstallerMetadata{Title: "x", BundleIdentifier: payload}, "bundle_identifier = '" + escaped + "';"},
		{"pkg bundle identifier", FullInstallerMetadata{Extension: "pkg", Title: "x", BundleIdentifier: payload}, "bundle_identifier = '" + escaped + "';"},
		{"msi upgrade code", FullInstallerMetadata{Extension: "msi", Title: "x", UpgradeCode: payload}, "upgrade_code = '" + escaped + "';"},
		{"msi product code", FullInstallerMetadata{Extension: "msi", Title: "x", PackageIDs: []string{payload}}, "identifying_number = '" + escaped + "';"},
		{"deb title", FullInstallerMetadata{Extension: "deb", Title: payload}, "name = '" + escaped + "' AND status"},
		{"rpm title", FullInstallerMetadata{Extension: "rpm", Title: payload}, "name = '" + escaped + "'\n);"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policyData, err := Generate(tc.meta)
			require.NoError(t, err)
			require.Contains(t, policyData.Query, tc.want)
			// Name and description aren't SQL and must keep the original value.
			require.NotContains(t, policyData.Name, "''")
		})
	}
}
