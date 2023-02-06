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

var versionRegExp, _ = regexp.Compile(`(?i)version \d+\.\d+(\.\d+)? \(?build \d+\)?`)

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

	z := html.NewTokenizer(reader)
	for {
		switch z.Next() {
		case html.ErrorToken:
			// If io.EOF, we are done...
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
							break
						}
					}
				}
			}

			// Release dates could also be in the form of
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
							}
						}
					}
				}
			}

			// The version could be inside a <em> element like
			// <em>Version 16.69.1 (Build 23011802)</em>
			if token.Data == "em" {
				// Check if the next text node contains a version string
				if z.Next() == html.TextToken {
					verToken := z.Token()
					if versionRegExp.MatchString(verToken.Data) {
						releases[len(releases)-1].Version = verToken.Data
					}
				}
			}
		}
	}
}
