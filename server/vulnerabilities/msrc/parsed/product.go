package parsed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Product abstracts a MS full product name.
// A full product name includes the name of the product plus its arch
// (if any) and its version (if any).
type Product string

type Products map[string]Product

var ErrNoMatch = errors.New("no product matches")

func (p Products) GetMatchForOS(ctx context.Context, os fleet.OperatingSystem) (string, error) {
	var dvMatch, noDvMatch string

	for pID, product := range p {
		normalizedOS := NewProductFromOS(os)
		if product.Name() != normalizedOS.Name() {
			continue
		}

		archMatch := product.Arch() == "all" || normalizedOS.Arch() == "all" || product.Arch() == normalizedOS.Arch()
		if !archMatch {
			continue
		}

		if product.HasDisplayVersion() && os.DisplayVersion != "" && strings.Contains(string(product), os.DisplayVersion) {
			dvMatch = pID
			break
		}

		// If os.DisplayVersion is empty, we need to confirm that the product
		// matches the correct build number. This is necessary to avoid false
		// positives when vulnerability scans have run before the host has been
		// updated after an upgrade to fleet v4.44.0 or later
		if !product.HasDisplayVersion() {
			var build string
			parts := strings.Split(os.KernelVersion, ".")
			if len(parts) > 3 {
				build = parts[2]
			}
			if build == "22000" || build == "10240" {
				noDvMatch = pID
			}
		}
	}

	if dvMatch == "" && noDvMatch == "" {
		return "", ctxerr.Wrap(ctx, ErrNoMatch)
	}

	if dvMatch == "" {
		return noDvMatch, nil
	}

	return dvMatch, nil
}

func NewProductFromFullName(fullName string) Product {
	// If the full name includes a version, return it as-is.
	p := Product(fullName)
	if p.HasDisplayVersion() {
		return p
	}

	// Several Windows products listed in MSRC bulletins don't include the OS version number.
	// We need this to match the product with a host's OS, so we'll add them here.
	versionString := ""
	switch {
	case strings.Contains(fullName, "Windows Server 2022"):
		versionString = "21H2"

	case strings.Contains(fullName, "Windows Server 2016"):
		versionString = "1607"

	case strings.Contains(fullName, "Windows Server 2019"):
		versionString = "1809"

	case strings.Contains(fullName, "Windows 8.1"):
		versionString = "6.3 / NT 6.3"

	case strings.Contains(fullName, "Windows RT 8.1"):
		versionString = "6.3 / NT 6.3"

	case strings.Contains(fullName, "Windows Server 2012 R2"):
		versionString = "6.3 / NT 6.3"

	case strings.Contains(fullName, "Windows Server 2012"):
		versionString = "6.2 / NT 6.2"

	case strings.Contains(fullName, "Windows Server 2008 R2"):
		versionString = "6.1 / NT 6.1"

	case strings.Contains(fullName, "Windows 7"):
		versionString = "6.1 / NT 6.1"

	case strings.Contains(fullName, "Windows Server 2008"):
		versionString = "6.0 / NT 6.0"
	}

	finalName := fullName
	if versionString != "" {
		finalName += (" Version " + versionString)
	}

	return Product(finalName)
}

func NewProductFromOS(os fleet.OperatingSystem) Product {
	return Product(fmt.Sprintf("%s for %s", os.Name, os.Arch))
}

// Arch returns the archicture for the current Microsoft product, if none can
// be found then "all" is returned. Returned values are meant to match the values returned from
// `SELECT arch FROM os_version` in OSQuery.
// eg:
// "Windows 10 Version 1803 for 32-bit Systems" => "32-bit"
func (p Product) Arch() string {
	val := string(p)
	switch {
	case strings.Contains(val, "ARM 64-bit") ||
		strings.Contains(val, "ARM64"):
		return "arm64"
	case strings.Contains(val, "x64") ||
		strings.Contains(val, "64-bit") ||
		strings.Contains(val, "x86_64"):
		return "64-bit"
	case strings.Contains(val, "32-bit") ||
		strings.Contains(val, "x86"):
		return "32-bit"
	case strings.Contains(val, "Itanium-Based"):
		return "itanium"
	default:
		return "all"
	}
}

// HasDisplayVersion returns true if the current Microsoft product
// has a display version in the name.
// Display Version refers to the version of the product that is
// displayed to the user: eg. 22H2
// Year/Half refers to the year and half of the year that the product
// was released: eg. 2nd Half of 2022
func (p Product) HasDisplayVersion() bool {
	keywords := []string{"version", "edition"}
	for _, k := range keywords {
		if strings.Contains(strings.ToLower(string(p)), k) {
			return true
		}
	}
	return false
}

// Name returns the name for the current Microsoft product, if none can
// be found then "" is returned.
// eg:
// "Windows 10 Version 1803 for 32-bit Systems" => "Windows 10"
// "Windows Server 2008 R2 for Itanium-Based Systems Service Pack 1" => "Windows Server 2008 R2"
func (p Product) Name() string {
	val := string(p)
	switch {
	// Desktop versions
	case strings.Contains(val, "Windows 7"):
		return "Windows 7"
	case strings.Contains(val, "Windows 8.1"):
		return "Windows 8.1"
	case strings.Contains(val, "Windows RT 8.1"):
		return "Windows RT 8.1"
	case strings.Contains(val, "Windows 10"):
		return "Windows 10"
	case strings.Contains(val, "Windows 11"):
		return "Windows 11"

	// Server versions
	case strings.Contains(val, "Windows Server 2008 R2"):
		return "Windows Server 2008 R2"
	case strings.Contains(val, "Windows Server 2012 R2"):
		return "Windows Server 2012 R2"

	case strings.Contains(val, "Windows Server 2008"):
		return "Windows Server 2008"
	case strings.Contains(val, "Windows Server 2012"):
		return "Windows Server 2012"
	case strings.Contains(val, "Windows Server 2016"):
		return "Windows Server 2016"
	case strings.Contains(val, "Windows Server 2019"):
		return "Windows Server 2019"
	case strings.Contains(val, "Windows Server 2022"):
		return "Windows Server 2022"
	case strings.Contains(val, "Windows Server,"):
		return "Windows Server"

	default:
		return ""
	}
}

// Matches checks whehter product A matches product B by checking to see if both are for the same
// product and if the architecture they target are compatible. This function is commutative.
func (p Product) Matches(o Product) bool {
	if p.Name() != o.Name() {
		return false
	}

	return p.Arch() == "all" || o.Arch() == "all" || p.Arch() == o.Arch()
}
