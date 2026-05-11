package xml

// FeedResult groups together products and their vulnerabilities.
type FeedResult struct {
	WinVulnerabilities []Vulnerability
	WinProducts        map[string]Product
}
