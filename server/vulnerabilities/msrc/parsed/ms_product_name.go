package msrc_parsed

import "strings"

// ArchFromMsFullProdName returns the archicture from a Microsoft full product name string, if none can
// be found then "all" is returned.
// eg:
// "Windows 10 Version 1803 for 32-bit Systems" => "32"
func ArchFromMsFullProdName(fullName string) string {
	switch {
	case strings.Index(fullName, "32-bit") != -1:
		return "32-bit"
	case strings.Index(fullName, "x64") != -1:
		return "64-bit"
	case strings.Index(fullName, "ARM64") != -1:
		return "arm64"
	case strings.Index(fullName, "Itanium-Based") != -1:
		return "itanium"
	default:
		return "all"
	}
}

// NameFromMsFullProdName returns the prod name from a Microsoft full product name string, if none can
// be found then "" is returned.
// eg:
// "Windows 10 Version 1803 for 32-bit Systems" => "Windows 10"
// "Windows Server 2008 R2 for Itanium-Based Systems Service Pack 1" => "Windows Server 2008 R2"
func NameFromMsFullProdName(fullName string) string {
	switch {
	// Desktop versions
	case strings.HasPrefix(fullName, "Windows 7"):
		return "Windows 7"
	case strings.HasPrefix(fullName, "Windows 8.1"):
		return "Windows 8.1"
	case strings.HasPrefix(fullName, "Windows RT 8.1"):
		return "Windows RT 8.1"
	case strings.HasPrefix(fullName, "Windows 10"):
		return "Windows 10"
	case strings.HasPrefix(fullName, "Windows 11"):
		return "Windows 11"

	// Server versions
	case strings.HasPrefix(fullName, "Windows Server 2008 R2"):
		return "Windows Server 2008 R2"
	case strings.HasPrefix(fullName, "Windows Server 2012 R2"):
		return "Windows Server 2012 R2"

	case strings.HasPrefix(fullName, "Windows Server 2008"):
		return "Windows Server 2008"
	case strings.HasPrefix(fullName, "Windows Server 2012"):
		return "Windows Server 2012"
	case strings.HasPrefix(fullName, "Windows Server 2016"):
		return "Windows Server 2016"
	case strings.HasPrefix(fullName, "Windows Server 2019"):
		return "Windows Server 2019"
	case strings.HasPrefix(fullName, "Windows Server 2022"):
		return "Windows Server 2022"
	case strings.HasPrefix(fullName, "Windows Server,"):
		return "Windows Server"

	default:
		return ""
	}
}
