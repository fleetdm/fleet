package macoffice

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	VersionPattern = regexp.MustCompile(`(?i)version (\d+\.\d+(\.\d+)?) \(?build \d+\)?`)
	cveLinkPattern = regexp.MustCompile(`CVE(-\d+)+$`)
)

var IdToType = map[string]ProductType{
	"office-suite": WholeSuite,
	"outlook":      Outlook,
	"word":         Word,
	"powerpoint":   PowerPoint,
	"excel":        Excel,
	"onenote":      OneNote,
}

var nameToType = map[string]ProductType{
	"office suite":         WholeSuite,
	"outlook":              Outlook,
	"word":                 Word,
	"powerpoint":           PowerPoint,
	"excel":                Excel,
	"onenote":              OneNote,
	"microsoft autoupdate": WholeSuite,
}

type releaseNotesURL string // type added to calm gosec variable URL warnings down
const (
	primaryURL = releaseNotesURL("https://learn.microsoft.com/en-us/officeupdates/release-notes-office-for-mac")
	archiveURL = releaseNotesURL("https://learn.microsoft.com/en-us/officeupdates/release-notes-office-for-mac-archived")
)

func GetReleaseNotes(includeInvalid bool) (ReleaseNotes, error) {
	var relNotes ReleaseNotes

	relNotes, err := addReleaseNotes(relNotes, primaryURL, includeInvalid)
	if err != nil {
		return nil, err
	}

	relNotes, err = addReleaseNotes(relNotes, archiveURL, includeInvalid)
	if err != nil {
		return nil, err
	}

	return relNotes, nil
}

func addReleaseNotes(relNotes ReleaseNotes, url releaseNotesURL, includeInvalid bool) (ReleaseNotes, error) {
	res, err := http.Get(string(url))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	parsed, err := parseReleaseHTML(res.Body)
	if err != nil {
		return nil, err
	}

	for _, rn := range parsed {
		// Under normal operation (outside parser tests), we only care about release notes that have a version set
		// (because we need that for matching software entries) and also that contain some
		// security updates (because we only intend to use the release notes for vulnerability processing).
		if includeInvalid || rn.Valid() {
			relNotes = append(relNotes, rn)
		}
	}

	return relNotes, nil
}

func parseRelDate(raw string) (time.Time, bool) {
	layouts := []string{"January-2-2006", "January-2-2006-release", "January 2, 2006"}
	for _, l := range layouts {
		relDate, err := time.Parse(l, raw)
		if err == nil {
			return relDate, true
		}
	}
	return time.Time{}, false
}

func parseVulnLink(token html.Token) (string, bool) {
	for _, a := range token.Attr {
		// Check if the link points to a CVE
		if a.Key == "href" && cveLinkPattern.MatchString(a.Val) {
			parts := strings.Split(a.Val, "/")
			return parts[len(parts)-1], true
		}
	}

	return "", false
}

// returns the id attr value of an html token, an empty string otherwise
func getId(token html.Token) string {
	for _, attr := range token.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}

	return ""
}

// ParseReleaseHTML parses the release page using the provided reader. It is assumed that elements
// in the page appear in order: first the release date then the version and finally any related
// security updates.
func parseReleaseHTML(reader io.Reader) ([]ReleaseNote, error) {
	var result []ReleaseNote

	// We use these pieces of state to keep track of whether we are inside a 'Security Updates'
	// section of a release and also what product the security updates applies to.
	var insideSecUpts bool
	var currentProduct ProductType

	z := html.NewTokenizer(reader)
	for {
		switch z.Next() {
		case html.ErrorToken:
			// If EOF we are done...
			if z.Err() == io.EOF {
				return result, nil
			}
			return nil, z.Err()
		case html.StartTagToken:
			token := z.Token()
			//
			//--------------
			// Release date
			// --------------
			//
			// The release date could be in a <h2> element like
			// <h2 id="january-19-2023" class="heading-anchor">January 19, 2023</h2>
			// or
			// <h2 id="november-12-2019-release" class="heading-anchor">November 12, 2019
			// release</h2>
			// Either way, we try to parse the id attribute for the release date.
			if token.Data == "h2" {
				if relDate, ok := parseRelDate(getId(token)); ok {
					result = append(result, ReleaseNote{Date: relDate})
					// Reset the state since we are inside a new release
					insideSecUpts = false
					break
				}
			}
			// The release date could also be in the form of
			// <strong>Release Date:</strong> January 11, 2017
			if token.Data == "strong" {
				// Check that the <strong> tag contains a 'Release Date:' text node
				if z.Next() == html.TextToken && z.Token().Data == "Release Date:" &&
					// The next token should be the closing tag
					z.Next() == html.EndTagToken && z.Token().Data == "strong" &&
					// And the next one should be the release date
					z.Next() == html.TextToken {
					if relDate, ok := parseRelDate(strings.TrimSpace(z.Token().Data)); ok {
						result = append(result, ReleaseNote{Date: relDate})
						// Reset state since we are inside a new release section
						insideSecUpts = false
					}
				}
			}

			//
			//--------------
			// Version
			// --------------
			//
			// Versions are always inside a <em> element like
			// <em>Version 16.69.1 (Build 23011802)</em>
			if token.Data == "em" {
				// Check if the text node that follows contains a proper version string
				if z.Next() == html.TextToken {
					t := z.Token()
					if VersionPattern.MatchString(t.Data) {
						result[len(result)-1].Version = t.Data
					}
				}
			}

			//
			//-----------------
			// Security updates
			// ----------------
			//
			// Security updates can be contained in a block that starts with a 'Security Updates' <h3> element:
			// <h3 id="security-updates-<some_id>" class="heading-anchor">Security updates</h3>
			//
			// Followed by one or more sub-sections that start with the Product name:
			// <h3 id="office-suite" class="heading-anchor">Office Suite</h3>
			//
			// Or a single <h3> element in the form of:
			// <h3 id="word-security-updates-1" class="heading-anchor">Word: Security updates</h3>
			//
			// Followed by a list of CVEs
			// <ul>
			// <li><a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2023-21734" data-linktype="external">CVE-2023-21734</a></li>
			// <li><a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2023-21735" data-linktype="external">CVE-2023-21735</a></li>
			// </ul>
			if token.Data == "h3" {
				id := getId(token)

				if strings.Contains(id, "security-updates") {
					insideSecUpts = true
				}

				for k, v := range IdToType {
					if strings.HasPrefix(id, k) && insideSecUpts {
						currentProduct = v
						break
					}
				}
			}

			// CVEs are defined as links like:
			// <a href="https://portal.msrc.microsoft.com/en-us/security-guidance/advisory/CVE-2019-1148" data-linktype="external">CVE-2019-1148</a>
			if insideSecUpts && token.Data == "a" {
				if cve, ok := parseVulnLink(token); ok {
					result[len(result)-1].AddSecurityUpdate(currentProduct, cve)
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
			if token.Data == "th" && !insideSecUpts {
				// Try to determine if we are inside a 'release table' (like the one above) by
				// looking at the th siblings and children
				switch z.Next() {
				case html.StartTagToken:
					insideSecUpts = z.Token().Data == "strong" &&
						z.Next() == html.TextToken &&
						z.Token().Data == "Application"

				case html.TextToken:
					insideSecUpts = z.Token().Data == "Application"
				}
			}

			if token.Data == "td" && insideSecUpts {
				z.Next()
				t := z.Token()

				// A table cell could contain either the "product" name ...
				if t.Type == html.TextToken {
					pName := strings.ToLower(strings.Trim(t.Data, " "))
					for k, v := range nameToType {
						if strings.HasPrefix(pName, k) {
							currentProduct = v
						}
					}
				}

				// Or a link to a CVE
				if t.Data == "a" {
					if cve, ok := parseVulnLink(t); ok {
						result[len(result)-1].AddSecurityUpdate(currentProduct, cve)
					}
				}
			}
		}
	}
}
