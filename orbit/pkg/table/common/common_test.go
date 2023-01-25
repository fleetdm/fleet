//go:build darwin
// +build darwin

package common

import (
	"testing"
)

func TestGetConsoleUidGid(t *testing.T) {
	_, _, err := GetConsoleUidGid()
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s`, err)
	}
}

func TestGetIntFromXMLWithTags(t *testing.T) {
	testXML := `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
  <title>W3Schools Home Page</title>
  <link>https://www.w3schools.com</link>
  <description>Free web building tutorials</description>
  <parentTag>
    <tag>tagValue</tag>
    <integer>11</integer>
  </parentTag>
  <item>
    <title>RSS Tutorial</title>
    <link>https://www.w3schools.com/xml/xml_rss.asp</link>
    <description>New RSS tutorial on W3Schools</description>
  </item>
  <item>
    <title>XML Tutorial</title>
    <link>https://www.w3schools.com/xml</link>
    <description>New XML tutorial on W3Schools</description>
  </item>
</channel>
</rss>`

	val, err := GetIntFromXMLWithTags(testXML, "parentTag", "tag", "tagValue")
	if err != nil {
		t.Fatalf(`Err expected to be nil. got %s`, err)
	}
	if val != "11" {
		t.Fatalf(`Val expected "11", got %s`, val)
	}

}
