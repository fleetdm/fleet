package parsed

import "strings"

// Product abstracts a MS full product name.
// A full product name includes the name of the product plus its arch
// (if any) and its version (if any).
type Product string

func NewProduct(fullName string) Product {
	return Product(fullName)
}

// Arch returns the archicture for the current Microsoft product, if none can
// be found then "all" is returned. Returned values are meant to match the values returned from
// `SELECT arch FROM os_version` in OSQuery.
// eg:
// "Windows 10 Version 1803 for 32-bit Systems" => "32-bit"
func (p Product) Arch() string {
	val := string(p)
	switch {
	case strings.Index(val, "32-bit") != -1:
		return "32-bit"
	case strings.Index(val, "x64") != -1:
		return "64-bit"
	case strings.Index(val, "ARM64") != -1:
		return "arm64"
	case strings.Index(val, "Itanium-Based") != -1:
		return "itanium"
	default:
		return "all"
	}
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
	case strings.Index(val, "Windows 7") != -1:
		return "Windows 7"
	case strings.Index(val, "Windows 8.1") != -1:
		return "Windows 8.1"
	case strings.Index(val, "Windows RT 8.1") != -1:
		return "Windows RT 8.1"
	case strings.Index(val, "Windows 10") != -1:
		return "Windows 10"
	case strings.Index(val, "Windows 11") != -1:
		return "Windows 11"

	// Server versions
	case strings.Index(val, "Windows Server 2008 R2") != -1:
		return "Windows Server 2008 R2"
	case strings.Index(val, "Windows Server 2012 R2") != -1:
		return "Windows Server 2012 R2"

	case strings.Index(val, "Windows Server 2008") != -1:
		return "Windows Server 2008"
	case strings.Index(val, "Windows Server 2012") != -1:
		return "Windows Server 2012"
	case strings.Index(val, "Windows Server 2016") != -1:
		return "Windows Server 2016"
	case strings.Index(val, "Windows Server 2019") != -1:
		return "Windows Server 2019"
	case strings.Index(val, "Windows Server 2022") != -1:
		return "Windows Server 2022"
	case strings.Index(val, "Windows Server,") != -1:
		return "Windows Server"

	default:
		return ""
	}
}
