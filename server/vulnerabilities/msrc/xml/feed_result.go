package xml

// FeedResult groups together products and their vulnerabilities.
type FeedResult struct {
	WinVulnerabities []Vulnerability
	WinProducts      map[string]Product
}
