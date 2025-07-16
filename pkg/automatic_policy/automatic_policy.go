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
type InstallerMetadata interface {
	PolicyName() (string, error)
	PolicyDescription() (string, error)
	PolicyQuery() (string, error)
	PolicyPlatform() (string, error)
}

type FMAInstallerMetadata struct {
	Title    string
	Platform string
	Query    string
}

func (m FMAInstallerMetadata) PolicyName() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	return fmt.Sprintf("[Install software] %s", m.Title), nil
}

func (m FMAInstallerMetadata) PolicyDescription() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	return fmt.Sprintf("Policy triggers automatic install of %s on each host that's missing this software.", m.Title), nil
}

func (m FMAInstallerMetadata) PolicyQuery() (string, error) {
	return m.Query, nil
}

func (m FMAInstallerMetadata) PolicyPlatform() (string, error) {
	return m.Platform, nil
}

type MacInstallerMetadata struct {
	BundleIdentifier string
	// Title is the software title extracted from a software package.
	Title string
}

func (m MacInstallerMetadata) PolicyName() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	return fmt.Sprintf("[Install software] %s", m.Title), nil
}

func (m MacInstallerMetadata) PolicyDescription() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	return fmt.Sprintf("Policy triggers automatic install of %s on each host that's missing this software.", m.Title), nil
}

func (m MacInstallerMetadata) PolicyQuery() (string, error) {
	if m.BundleIdentifier == "" {
		return "", ErrMissingBundleIdentifier
	}
	return fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", m.BundleIdentifier), nil
}

func (m MacInstallerMetadata) PolicyPlatform() (string, error) {
	return "darwin", nil
}

type FullInstallerMetadata struct {
	// BundleIdentifier is the bundle identifier for 'pkg' packages
	BundleIdentifier string

	// Title is the software title extracted from a software package.
	Title string

	// Extension is the extension of the software package.
	Extension string

	// PackageIDs contains the product code for 'msi' packages.
	PackageIDs []string
}

func (m FullInstallerMetadata) PolicyName() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	if m.Extension == "" {
		return "", ErrExtensionNotSupported
	}
	return fmt.Sprintf("[Install software] %s (%s)", m.Title, m.Extension), nil
}

func (m FullInstallerMetadata) PolicyDescription() (string, error) {
	if m.Title == "" {
		return "", ErrMissingTitle
	}
	description := fmt.Sprintf("Policy triggers automatic install of %s on each host that's missing this software.", m.Title)
	if m.Extension == "deb" || m.Extension == "rpm" {
		basedPrefix := "RPM"
		if m.Extension == "rpm" {
			basedPrefix = "Debian"
		}
		description += fmt.Sprintf(
			"\nSoftware won't be installed on Linux hosts with %s-based distributions because this policy's query is written to always pass on these hosts.",
			basedPrefix,
		)
	}

	return description, nil
}

func (m FullInstallerMetadata) PolicyQuery() (string, error) {
	switch m.Extension {
	case "pkg":
		if m.BundleIdentifier == "" {
			return "", ErrMissingBundleIdentifier
		}
		return fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", m.BundleIdentifier), nil
	case "msi":
		if len(m.PackageIDs) == 0 || m.PackageIDs[0] == "" {
			return "", ErrMissingProductCode
		}
		return fmt.Sprintf("SELECT 1 FROM programs WHERE upgrade_code = '%s';", m.PackageIDs[0]), nil
	case "deb":
		return fmt.Sprintf(
			// First inner SELECT will mark the policies as successful on non-DEB-based hosts.
			`SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM deb_packages) = 0
) OR EXISTS (
	SELECT 1 FROM deb_packages WHERE name = '%s'
);`, m.Title,
		), nil
	case "rpm":
		return fmt.Sprintf(
			// First inner SELECT will mark the policies as successful on non-RPM-based hosts.
			`SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM rpm_packages) = 0
) OR EXISTS (
	SELECT 1 FROM rpm_packages WHERE name = '%s'
);`, m.Title), nil
	default:
		return "", ErrExtensionNotSupported
	}
}

func (m FullInstallerMetadata) PolicyPlatform() (string, error) {
	switch m.Extension {
	case "pkg":
		return "darwin", nil
	case "msi":
		return "windows", nil
	case "deb":
		return "linux", nil
	case "rpm":
		return "linux", nil
	default:
		return "", ErrExtensionNotSupported
	}
}

var (
	// ErrExtensionNotSupported is returned if the extension is not supported to generate automatic policies.
	ErrExtensionNotSupported = errors.New("extension not supported")
	// ErrMissingBundleIdentifier is returned if the software extension is "pkg" and a bundle identifier was not extracted from the installer.
	ErrMissingBundleIdentifier = errors.New("missing bundle identifier")
	// ErrMissingProductCode is returned if the software extension is "msi" and a product code was not extracted from the installer.
	ErrMissingProductCode = errors.New("missing product code")
	// ErrMissingTitle is returned if a title was not extracted from the installer.
	ErrMissingTitle = errors.New("missing title")
)

// Generate generates the "trigger policy" from the metadata of a software package.
func Generate(metadata InstallerMetadata) (*PolicyData, error) {
	name, err := metadata.PolicyName()
	if err != nil {
		return nil, err
	}
	query, err := metadata.PolicyQuery()
	if err != nil {
		return nil, err
	}
	platform, err := metadata.PolicyPlatform()
	if err != nil {
		return nil, err
	}
	description, err := metadata.PolicyDescription()
	if err != nil {
		return nil, err
	}

	return &PolicyData{Name: name, Query: query, Description: description, Platform: platform}, nil
}
