// Package automatic_policy generates "trigger policies" from metadata of software packages.
package automatic_policy

import (
	"errors"
	"fmt"
)

// PolicyData contains generated data for a policy to trigger installation of a software package.
type PolicyData struct {
	// Name is the generated name of the policy.
	Name string
	// Query is the generated SQL/sqlite of the policy.
	Query string
	// Description is the generated description for the policy.
	Description string
	// Platform is the target platform for the policy.
	Platform string
}

// InstallerMetadata contains the metadata of a software package used to generate the policies.
type InstallerMetadata struct {
	// Title is the software title extracted from a software package.
	Title string
	// Extension is the extension of the software package.
	Extension string
	// BundleIdentifier contains the bundle identifier for 'pkg' packages.
	BundleIdentifier string
	// PackageIDs contains the product code for 'msi' packages.
	PackageIDs []string
}

var (
	// ErrExtensionNotSupported is returned if the extension is not supported to generate automatic policies.
	ErrExtensionNotSupported = errors.New("extension not supported")
	// ErrMissingBundleIdentifier is returned if the software extension is "pkg" and a bundle identifier was not extracted from the installer.
	ErrMissingBundleIdentifier = errors.New("missing bundle identifier")
	// ErrMissingProductCode is returned if the software extension is "msi" and a product code was not extracted from the installer
	ErrMissingProductCode = errors.New("missing product code")
)

// Generate generates the "trigger policy" from the metadata of a software package.
func Generate(metadata InstallerMetadata) (*PolicyData, error) {
	if metadata.Extension != "pkg" && metadata.Extension != "msi" && metadata.Extension != "deb" && metadata.Extension != "rpm" {
		return nil, ErrExtensionNotSupported
	}

	if metadata.Extension == "pkg" && metadata.BundleIdentifier == "" {
		return nil, ErrMissingBundleIdentifier
	}

	if metadata.Extension == "msi" && (len(metadata.PackageIDs) == 0 || metadata.PackageIDs[0] == "") {
		return nil, ErrMissingProductCode
	}

	name := fmt.Sprintf("[Install software] %s (%s)", metadata.Title, metadata.Extension)

	description := fmt.Sprintf("Policy triggers automatic install of %s on each host that's missing this software.", metadata.Title)
	if metadata.Extension == "deb" || metadata.Extension == "rpm" {
		basedPrefix := "Debian"
		if metadata.Extension == "rpm" {
			basedPrefix = "RPM"
		}
		description += fmt.Sprintf(
			"\nSoftware won't be installed on Linux hosts with %s-based distributions because this policy's query is written to always pass on these hosts.",
			basedPrefix,
		)
	}

	switch metadata.Extension {
	case "pkg":
		return &PolicyData{
			Name:        name,
			Query:       fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", metadata.BundleIdentifier),
			Platform:    "darwin",
			Description: description,
		}, nil
	case "msi":
		return &PolicyData{
			Name:        name,
			Query:       fmt.Sprintf("SELECT 1 FROM programs WHERE identifying_number = '%s';", metadata.PackageIDs[0]),
			Platform:    "windows",
			Description: description,
		}, nil
	case "deb":
		return &PolicyData{
			Name: name,
			Query: fmt.Sprintf(
				// First inner SELECT will mark the policies as successful on RHEL hosts.
				`SELECT 1 WHERE EXISTS (
	SELECT 1 FROM os_version WHERE platform = 'rhel'
) OR EXISTS (
	SELECT 1 FROM deb_packages WHERE name = '%s'
);`, metadata.Title,
			),
			Platform:    "linux",
			Description: description,
		}, nil
	case "rpm":
		return &PolicyData{
			Name: name,
			Query: fmt.Sprintf(
				// First inner SELECT will mark the policies as successful on non-RHEL-based hosts.
				`SELECT 1 WHERE EXISTS (
	SELECT 1 FROM os_version WHERE platform != 'rhel'
) OR EXISTS (
	SELECT 1 FROM rpm_packages WHERE name = '%s'
);`, metadata.Title),
			Platform:    "linux",
			Description: description,
		}, nil
	default:
		return nil, ErrExtensionNotSupported
	}
}
