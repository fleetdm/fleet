package apple_mdm

import (
	"net/url"
	"path"
)

// DEPName is the identifier/name used in nanodep MySQL storage which
// holds the DEP configuration.
//
// Fleet uses only one DEP configuration set for the whole deployment.
const DEPName = "fleet"

const (
	// SCEPPath is Fleet's HTTP path for the SCEP service.
	SCEPPath = "/mdm/apple/scep"
	// MDMPath is Fleet's HTTP path for the core MDM service.
	MDMPath = "/mdm/apple/mdm"

	// EnrollPath is the HTTP path that serves the mobile profile to devices when enrolling.
	EnrollPath = "/api/mdm/apple/enroll"
	// InstallerPath is the HTTP path that serves installers to Apple devices.
	InstallerPath = "/api/mdm/apple/installer"
)

func ResolveAppleMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, MDMPath)
}

func ResolveAppleEnrollMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, EnrollPath)
}

func ResolveAppleSCEPURL(serverURL string) (string, error) {
	return resolveURL(serverURL, SCEPPath)
}

func resolveURL(serverURL, relPath string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, relPath)
	return u.String(), nil
}
