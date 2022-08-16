package msrc_input

// ResultXML groups together products and their vulnerabilities.
type ResultXML struct {
	WinVulnerabities []VulnerabilityXML
	WinProducts      map[string]ProductXML
}
