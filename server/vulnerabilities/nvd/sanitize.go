package nvd

import (
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var nonAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var sanitizeVersionRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

var stopWords = map[string]bool{
	".":            true,
	"THE":          true,
	"The":          true,
	"Inc":          true,
	"Inc.":         true,
	"Incorporated": true,
	"Corporation":  true,
	"Corp":         true,
	"Foundation":   true,
	"Software":     true,
	"com":          true,
	"org":          true,
}

var langCodes = map[string]bool{
	"af-ZA": true,
	"bg-BG": true,
	"ca-AD": true,
	"cs-CZ": true,
	"cy-GB": true,
	"da-DK": true,
	"de-DE": true,
	"el-GR": true,
	"en-US": true,
	"es-ES": true,
	"et-EE": true,
	"fa-IR": true,
	"fi-FI": true,
	"fr-FR": true,
	"he-IL": true,
	"hi-IN": true,
	"hr-HR": true,
	"hu-HU": true,
	"id-ID": true,
	"is-IS": true,
	"it-IT": true,
	"ja-JP": true,
	"km-KH": true,
	"ko-KR": true,
	"lt-LT": true,
	"lv-LV": true,
	"mn-MN": true,
	"nb-NO": true,
	"nl-NL": true,
	"nn-NO": true,
	"pl-PL": true,
	"pt-PT": true,
	"ro-RO": true,
	"ru-RU": true,
	"sk-SK": true,
	"sl-SI": true,
	"sr-RS": true,
	"sv-SE": true,
	"th-TH": true,
	"tr-TR": true,
	"uk-UA": true,
	"vi-VN": true,
	"zh-CN": true,
}

// sanitizeSoftwareName sanitizes the software.Name by:
// - Removing any arch string contained in the name
// - Removing any language code
// - Removing any general remarks (for example: 7-zip - The best software)
// - Removing the '.app' suffix
// - Removing any '()' and its contents
// - Removing any extra spaces
// - Lowercasing the name
// - Removing parts from the bundle identifier
// - Removing version contained in homebrew_packages name
func sanitizeSoftwareName(s *fleet.Software) string {
	archs := regexp.MustCompile(` \(?x64\)?|\(?64-bit\)?|\(?64bit\)?|\(?amd64\)? `)
	ver := regexp.MustCompile(` \.?\(?(\d+\.)?(\d+\.)?(\*|\d+)\)?\s?`)
	gen := regexp.MustCompile(` \(\w+\)\s?`)
	comments := regexp.MustCompile(` (-|:)\s?.+`)
	versions := regexp.MustCompile(`@\d+($|(\.\d+($|\..+)))`) // @3 or @3.9 or @3.9.18 or @3.9.18_2

	r := strings.ToLower(s.Name)
	r = strings.TrimSuffix(r, ".app")

	// Remove vendor, for 'apps' the vendor name is usually after the top level domain part.
	r = strings.ReplaceAll(r, strings.ToLower(s.Vendor), "")
	bundleParts := strings.Split(s.BundleIdentifier, ".")
	if len(bundleParts) > 2 {
		r = strings.ReplaceAll(r, strings.ToLower(bundleParts[1]), "")
	}

	if len(r) == 0 {
		r = strings.ToLower(s.Name)
		r = strings.TrimSuffix(r, ".app")
	}

	r = archs.ReplaceAllString(r, "")
	r = ver.ReplaceAllString(r, "")
	r = gen.ReplaceAllString(r, "")

	r = strings.ReplaceAll(r, "—", "-")
	r = strings.ReplaceAll(r, "–", "-")
	r = comments.ReplaceAllString(r, "")

	for l := range langCodes {
		ln := strings.ToLower(l)
		r = strings.ReplaceAll(r, ln, "")
	}

	r = strings.ReplaceAll(r, "(", " ")
	r = strings.ReplaceAll(r, ")", " ")
	r = strings.Join(strings.Fields(r), " ")

	// Remove @<version> from homebrew names
	if s.Source == "homebrew_packages" {
		r = versions.ReplaceAllString(r, "")
	}

	return r
}

func productVariations(s *fleet.Software) []string {
	var r []string
	rSet := make(map[string]bool)

	sn := sanitizeSoftwareName(s)

	withoutVendorParts := sn
	for _, p := range strings.Split(s.Vendor, " ") {
		pL := strings.ToLower(p)
		withoutVendorParts = strings.Join(strings.Fields(strings.ReplaceAll(withoutVendorParts, pL, "")), " ")
	}
	if withoutVendorParts != "" {
		rSet[strings.ReplaceAll(withoutVendorParts, " ", "")] = true
		rSet[strings.ReplaceAll(withoutVendorParts, " ", "_")] = true
	}

	rSet[strings.ReplaceAll(sn, " ", "_")] = true
	rSet[strings.ReplaceAll(sn, " ", "")] = true

	for re := range rSet {
		r = append(r, re)
	}

	// VSCode extensions have a unique s.Name of the form "<vendor>.<extension>" (aka extension ID)
	if s.Source == "vscode_extensions" {
		parts := strings.SplitN(s.Name, ".", 2)
		if len(parts) == 2 && parts[1] != "" {
			r = append(r, parts[1])
		}
	}

	return r
}

func vendorVariations(s *fleet.Software) []string {
	var r []string
	rSet := make(map[string]bool)

	if s.Vendor == "" && s.BundleIdentifier == "" {
		return r
	}

	if s.Vendor != "" {
		for _, v := range strings.Split(s.Vendor, " ") {
			if !stopWords[v] {
				rSet[strings.ToLower(v)] = true
			}
		}
		rSet[strings.ToLower(strings.ReplaceAll(s.Vendor, " ", "_"))] = true
		rSet[strings.ToLower(strings.ReplaceAll(s.Vendor, " ", ""))] = true
	}

	for _, v := range strings.Split(s.BundleIdentifier, ".") {
		if !stopWords[v] {
			rSet[strings.ToLower(v)] = true
		}
	}

	for re := range rSet {
		if re != "" {
			r = append(r, re)
		}
	}

	// VSCode extensions have a unique s.Name of the form "<vendor>.<extension>" (aka extension ID)
	if s.Source == "vscode_extensions" {
		parts := strings.SplitN(s.Name, ".", 2)
		if len(parts) == 2 && parts[0] != "" {
			r = append(r, parts[0])
		}
	}

	return r
}

// sanitizeMatch sanitizes the search string for sqlite fts queries. Replaces all non alpha numeric characters with spaces.
func sanitizeMatch(s string) string {
	s = strings.TrimSuffix(s, ".app")
	s = nonAlphaNumeric.ReplaceAllString(s, " ")
	return s
}

// sanitizeVersion attempts to sanitize versions and attempt to make it dot separated.
// Eg Zoom reports version as "5.11.1 (8356)". In the NVD CPE dictionary it should be 5.11.1.8356.
func sanitizeVersion(version string) string {
	parts := sanitizeVersionRe.Split(version, -1)
	return strings.Trim(strings.Join(parts, "."), ".")
}

func targetSW(s *fleet.Software) string {
	switch s.Source {
	case "apps":
		return "macos"
	case "homebrew_packages":
		return "macos" // osquery homebrew_packages table is currently only for macOS (2024/08/12)
	case "python_packages":
		return "python"
	case "chrome_extensions":
		return "chrome"
	case "firefox_addons":
		return "firefox"
	case "safari_extensions":
		return "safari"
	case "npm_packages":
		return `node.js`
	case "programs":
		return "windows"
	case "vscode_extensions":
		return "visual_studio_code"
	}
	return "*"
}
