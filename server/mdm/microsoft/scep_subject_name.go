package microsoft_mdm

import (
	"encoding/xml"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// Windows parses the SCEP SubjectName as an X.500 string: ",", "+" and ";" separate attributes, and
// a value holding any of them must be quoted across its whole length. Verified on Windows 11:
// CertEnroll rejects CN=user+idp@example.com and the RFC 4514 form CN=user\+idp@example.com, a partly
// quoted value fails too, and CN=alice;O=Fleet QA parses as two attributes. Unlike RFC 4514 a
// backslash is not an escape here, it is literal inside quotes and out; a quote inside a quoted
// value is written as "".
//
// Attribute boundaries are only unambiguous before substitution, so every value about to receive one
// is quoted then, whatever it turns out to hold. Quoting a value that needed no quoting parses
// identically, and deciding per value would mean recovering boundaries that substitution has
// already destroyed. A second pass afterwards doubles any quote that arrived inside those quotes.

const (
	cdataOpen  = "<![CDATA["
	cdataClose = "]]>"
)

// mapDNAttributes applies transform to each "KEY=value" attribute of the DN and puts the separators
// back. A separator inside a quoted value is content, not a boundary, so both passes below see the
// same attributes.
func mapDNAttributes(dn string, transform func(attribute string) string) string {
	var b strings.Builder
	b.Grow(len(dn))
	for rest := dn; ; {
		i := indexDNSeparator(rest)
		if i < 0 {
			b.WriteString(transform(rest))
			return b.String()
		}
		b.WriteString(transform(rest[:i]))
		b.WriteByte(rest[i])
		rest = rest[i+1:]
	}
}

// quoteAttributeValue is the first pass: it quotes the value of an attribute that has something to
// substitute into it, unless the admin already quoted it.
func quoteAttributeValue(attribute string) string {
	key, value, found := strings.Cut(attribute, "=")
	if !found || key == "" || value == "" || !carriesSubstitution(value) {
		return attribute
	}
	if trimmed := strings.TrimSpace(value); !isQuoted(trimmed) {
		return key + `="` + trimmed + `"`
	}
	return attribute
}

// doubleAttributeQuotes is the second pass: it doubles any quote inside a quoted value. Every
// substitution into profile XML is escaped, so a substituted quote arrives as an entity and the raw
// quotes bounding each value are only ever Fleet's or the admin's.
func doubleAttributeQuotes(attribute string) string {
	key, value, found := strings.Cut(attribute, "=")
	trimmed := strings.TrimSpace(value)
	if !found || !isQuoted(trimmed) {
		return attribute
	}
	return key + `="` + quoteDoubler.Replace(trimmed[1:len(trimmed)-1]) + `"`
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

// indexDNSeparator returns the offset of the first ",", "+" or ";" that falls outside a quoted
// value, or -1. An unterminated quote hides everything after it.
func indexDNSeparator(dn string) int {
	inQuotes := false
	for i := 0; i < len(dn); i++ {
		switch dn[i] {
		case '"':
			inQuotes = !inQuotes
		case ',', '+', ';':
			if !inQuotes {
				return i
			}
		}
	}
	return -1
}

// carriesSubstitution reports whether Fleet will replace something in this value. Host vitals count:
// like Fleet variables they are XML-escaped, but neither carries X.500 escaping.
func carriesSubstitution(value string) bool {
	return variables.Find(value) != nil || fleet.FindCustomHostVitalIDs(value) != nil
}

// quoteDoubler doubles a quote inside a quoted value. Only the entity spellings are covered: a raw
// quote cannot appear inside a value, since that is what bounds it.
var quoteDoubler = strings.NewReplacer(
	"&#34;", "&#34;&#34;",
	"&quot;", "&quot;&quot;",
)

// transformSCEPSubjectNameData applies transformAttribute to each attribute of every SCEP
// SubjectName <Data>, leaving every other byte of the profile untouched.
func transformSCEPSubjectNameData(profileContents string, transformAttribute func(attribute string) string) string {
	if !strings.Contains(profileContents, fleet.WindowsSCEPSubjectNameSuffix) {
		return profileContents
	}
	spans, err := scepSubjectNameDataSpans(profileContents)
	if err != nil {
		// Upload validation already rejects unparseable XML; don't guess at its structure.
		return profileContents
	}
	var b strings.Builder
	b.Grow(len(profileContents))
	var prev int64
	for _, span := range spans {
		lead, dn, trail := unwrapCDATA(profileContents[span.start:span.end])
		b.WriteString(profileContents[prev:span.start])
		b.WriteString(lead + mapDNAttributes(dn, transformAttribute) + trail)
		prev = span.end
	}
	b.WriteString(profileContents[prev:])
	return b.String()
}

// unwrapCDATA splits content into the text of a single CDATA section and everything around it, so
// the text can be rewritten and the rest reassembled byte for byte. Content that is not one section
// surrounded only by whitespace comes back whole as the text. The wrapper is kept out of the DN so
// its terminator is not mistaken for part of the last attribute value.
func unwrapCDATA(content string) (lead, text, trail string) {
	body := strings.TrimSpace(content)
	if !strings.HasPrefix(body, cdataOpen) || !strings.HasSuffix(body, cdataClose) {
		return "", content, ""
	}
	text = body[len(cdataOpen) : len(body)-len(cdataClose)]
	// "]]>" is the one thing that cannot appear inside a section, so finding it means two sections.
	if strings.Contains(text, cdataClose) {
		return "", content, ""
	}
	lead, trail, _ = strings.Cut(content, body)
	return lead + cdataOpen, text, cdataClose + trail
}

// dataSpan is the half-open byte range of a <Data> element's content within the profile.
type dataSpan struct {
	start, end int64
}

// scepSubjectNameDataSpans returns the content span of every <Data> under an item targeting a SCEP
// SubjectName. Decoding rather than pattern-matching ignores a <Data> inside a comment or CDATA and
// pairs each one with its own item's LocURI whatever order they appear in.
func scepSubjectNameDataSpans(profileContents string) ([]dataSpan, error) {
	dec := xml.NewDecoder(strings.NewReader(profileContents))
	stack := make([]string, 0, 8)
	var (
		spans     []dataSpan
		locURI    string
		itemSpans []dataSpan
		dataStart int64
		// Trails the decoder by one token, so at an end element it marks where the content ended.
		prevOffset int64
	)
	inItem := func() bool { return slices.Contains(stack, "Item") }
	for {
		offsetBeforeToken := prevOffset
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return spans, nil
			}
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			switch t.Name.Local {
			case "Item":
				locURI = ""
				itemSpans = nil
			case "Data":
				if inItem() {
					dataStart = dec.InputOffset()
				}
			}
		case xml.CharData:
			if inItem() && inTargetLocURI(stack) {
				locURI += string(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "Item":
				if isSCEPSubjectNameLocURI(locURI) {
					spans = append(spans, itemSpans...)
				}
			case "Data":
				if inItem() {
					itemSpans = append(itemSpans, dataSpan{start: dataStart, end: offsetBeforeToken})
				}
			}
			// The decoder is strict, so every end element closes the element on top of the stack.
			stack = stack[:len(stack)-1]
		}
		prevOffset = dec.InputOffset()
	}
}

// isSCEPSubjectNameLocURI canonicalizes first so every scope spelling hits. The suffix excludes
// SubjectAlternativeNames, which is a ";"-separated list rather than a DN.
func isSCEPSubjectNameLocURI(locURI string) bool {
	canonical := fleet.CanonicalizeSCEPScope(locURI)
	return fleet.IsWindowsSCEPLocURI(canonical) &&
		strings.HasSuffix(canonical, fleet.WindowsSCEPSubjectNameSuffix)
}

func inTargetLocURI(stack []string) bool {
	n := len(stack)
	return n >= 2 && stack[n-2] == "Target" && stack[n-1] == "LocURI"
}
