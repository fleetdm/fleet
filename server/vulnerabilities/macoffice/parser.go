package macoffice

import (
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// TODO: Move this
const url = "https://learn.microsoft.com/en-us/officeupdates/release-notes-office-for-mac"

var (
	verPattern, _     = regexp.Compile(`(?i)version \d+\.\d+(\.\d+)? \(?build \d+\)?`)
	cveLinkPattern, _ = regexp.Compile(`CVE(-\d+)+$`)
)

var productIdMap = map[string]ProductType{
	"office-suite": OfficeSuite,
	"outlook":      Outlook,
	"word":         Word,
	"powerpoint":   PowerPoint,
	"excel":        Excel,
	"onenote":      OneNote,
}

func tryParsingReleaseDate(raw string) *time.Time {
	layouts := []string{"January-2-2006", "January-2-2006-release", "January 2, 2006"}
	for _, l := range layouts {
		relDate, err := time.Parse(l, raw)
		if err == nil {
			return &relDate
		}
	}
	return nil
}

// ParseReleaseHTML parses the release page using the provided reader.
func ParseReleaseHTML(reader io.Reader) ([]OfficeRelease, error) {
	var releases []OfficeRelease

	// State to keep track of the security updates
	var insideSecUpdatesSec bool
	var currentProduct *ProductType

	z := html.NewTokenizer(reader)
	for {
		switch z.Next() {
		case html.ErrorToken:
			// If EOF we are done...
			if z.Err() == io.EOF {
				return releases, nil
			}
			return nil, z.Err()
		case html.StartTagToken:
			token := z.Token()

			// The release date could be in a <h2> element like
			// <h2 id="january-19-2023" class="heading-anchor">January 19, 2023</h2>
			// or
			// <h2 id="november-12-2019-release" class="heading-anchor">November 12, 2019
			// release</h2>
			// In which case we try to parse the id attribute for the release date.
			if token.Data == "h2" {
				for _, attr := range token.Attr {
					if attr.Key == "id" {
						relDate := tryParsingReleaseDate(attr.Val)
						if relDate != nil {
							releases = append(releases, OfficeRelease{Date: *relDate})
							// Reset state since we are inside a new release section
							insideSecUpdatesSec = false
							currentProduct = nil
							break
						}
					}
				}
			}
			// The release date could also be in the form of
			// <p><strong>Release Date:</strong> January 11, 2017</p>
			if token.Data == "strong" {
				// Check that the <strong> tag contains a 'Release Date:' text node
				if z.Next() == html.TextToken && z.Token().Data == "Release Date:" {
					// The next token should be the closing tag
					if z.Next() == html.EndTagToken && z.Token().Data == "strong" {
						// And the next one should be the release date
						if z.Next() == html.TextToken {
							relDate := tryParsingReleaseDate(strings.TrimSpace(z.Token().Data))
							if relDate != nil {
								releases = append(releases, OfficeRelease{Date: *relDate})
								// Reset state since we are inside a new release section
								insideSecUpdatesSec = false
								currentProduct = nil
							}
						}
					}
				}
			}

			// The version is inside a <em> element like
			// <em>Version 16.69.1 (Build 23011802)</em>
			if token.Data == "em" {
				// Check if the text node that follows contains a proper version string
				if z.Next() == html.TextToken {
					t := z.Token()
					if verPattern.MatchString(t.Data) {
						releases[len(releases)-1].Version = t.Data
					}
				}
			}

			// Security updates can be contained in a block that starts with a 'Security Updates' <h3> element:
			// <h3 id="security-updates-<some_id>" class="heading-anchor">Security updates</h3>
			//
			// Followed by one or more sub-sections that start with the Product name:
			// <h3 id="office-suite" class="heading-anchor">Office Suite</h3>
			//
			// Followed by a list of CVEs
			// <ul>
			// <li><a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2023-21734" data-linktype="external">CVE-2023-21734</a></li>
			// <li><a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2023-21735" data-linktype="external">CVE-2023-21735</a></li>
			// </ul>
			if token.Data == "h3" {
				for _, a := range token.Attr {
					if a.Key == "id" {
						if strings.HasPrefix(a.Val, "security-updates") {
							insideSecUpdatesSec = true
							break
						}

						if p, ok := productIdMap[a.Val]; insideSecUpdatesSec && ok {
							currentProduct = &p
							break
						}
					}
				}
			}

			// Security vulnerabilities are contained in links
			// <a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2019-1148" data-linktype="external">CVE-2019-1148</a>
			if insideSecUpdatesSec && currentProduct != nil && token.Data == "a" {
				for _, a := range token.Attr {
					// Check if the link points to a CVE
					if a.Key == "href" {
						if cveLinkPattern.MatchString(a.Val) {
							parts := strings.Split(a.Val, "/")
							cve := parts[len(parts)-1]
							releases[len(releases)-1].SecurityUpdates = append(
								releases[len(releases)-1].SecurityUpdates,
								SecurityUpdate{
									Product:       *currentProduct,
									Vulnerability: cve,
								},
							)
						}
					}
				}
			}

			// Security updates could also in a table form:
			// <table aria-label="Table 1" class="table table-sm">
			// <thead>
			// <tr>
			// <th style="text-align: left;"><strong>Application</strong></th>
			// <th style="text-align: left;"><strong>Update</strong></th>
			// <th style="text-align: left;"><strong>Security updates</strong></th>
			// <th style="text-align: left;"><strong>Download link for update package</strong></th>
			// </tr>
			// </thead>
			// <tbody>
			// <tr>
			// <td style="text-align: left;">Word  <br><br></td>
			// <td style="text-align: left;"><strong>See your email attachments:</strong> Your email attachments are now available in the Shared tab.</td>
			// <td style="text-align: left;"><a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2019-0953" data-linktype="external">CVE-2019-0953</a>: Microsoft Word Remote Code Execution Vulnerability<br></td>
			// <td style="text-align: left;"><a href="https://officecdn.microsoft.com/pr/C1297A47-86C4-4C1F-97FA-950631F94777/MacAutoupdate//Microsoft_Word_16.25.19051201_Updater.pkg" data-linktype="external">Word update package</a><br></td>
			// </tr>
			// <tr>
			// ....
			// </tbody>
			// </table>
			//
			// Try to determine if we are inside a 'release' table
			if token.Data == "th" {
				seq := []string{
					"strong",
					"Application",
					"strong",
					"th",
					"th",
					"strong",
					"Update",
					"strong",
					"th",
				}
				// Check that the <strong> tag contains a 'Release Date:' text node
				if z.Next() == html.TextToken && z.Token().Data == "Release Date:" {
					// The next token should be the closing tag
					if z.Next() == html.EndTagToken && z.Token().Data == "strong" {
					}
				}
			}
		}
	}
}
