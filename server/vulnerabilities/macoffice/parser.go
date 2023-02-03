package macoffice

import (
	"io"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// TODO: Move this
const url = "https://learn.microsoft.com/en-us/officeupdates/release-notes-office-for-mac"

func tryParsingReleaseDate(rawVal string) *time.Time {
	layouts := []string{"January-2-2006", "January-2-2006-release", "January 2, 2006"}
	for _, l := range layouts {
		relDate, err := time.Parse(l, rawVal)
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
		if z.Next() == html.ErrorToken {
			// If io.EOF, we are done...
			if z.Err() == io.EOF {
				return releases, nil
			}
			return nil, z.Err()
		}

		token := z.Token()
		switch token.Type {
		case html.ErrorToken:
			return nil, z.Err()
		case html.StartTagToken:
			// The release date could be in a h2 element like
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
				// Check that the <strong> tag contains 'Release Date:'
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
		}
	}
}
