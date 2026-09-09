package microsoft_mdm

import (
	"encoding/xml"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"

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
//
// Windows reads the decoded profile, so both passes read a character reference the way Windows will,
// with one exception that separates Fleet's writing from the admin's: a raw quote bounds a value,
// and an escaped one is content. Fleet only ever substitutes an escaped quote, so the second pass
// can double what it substituted and leave what the admin wrote alone.

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
		i, width := indexDNSeparator(rest)
		if i < 0 {
			b.WriteString(transform(rest))
			return b.String()
		}
		b.WriteString(transform(rest[:i]))
		// Written back as the admin spelled it, character reference and all.
		b.WriteString(rest[i : i+width])
		rest = rest[i+width:]
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
	return key + `="` + doubleSubstitutedQuotes(trimmed[1:len(trimmed)-1]) + `"`
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

// indexDNSeparator returns the offset and byte length of the first ",", "+" or ";" that falls
// outside a quoted value, or -1. An unterminated quote hides everything after it.
//
// Windows decodes the profile before parsing the DN, so a separator can be spelled as a character
// reference and a reference's own ";" is not one. Both passes read references the same way, so the
// attributes they see stay the same across the substitution between them.
func indexDNSeparator(dn string) (offset, width int) {
	inQuotes := false
	for i := 0; i < len(dn); {
		if n := xmlRefLen(dn[i:]); n > 0 {
			if !inQuotes && isDNSeparator(xmlRefASCII(dn[i:i+n])) {
				return i, n
			}
			// A reference for a quote stays content: only a raw quote bounds a value, which is
			// what lets the pass below tell a substituted quote from the quoting around it.
			i += n
			continue
		}
		switch {
		case dn[i] == '"':
			inQuotes = !inQuotes
		case !inQuotes && isDNSeparator(dn[i]):
			return i, 1
		}
		i++
	}
	return -1, 0
}

func isDNSeparator(c byte) bool {
	return c == ',' || c == '+' || c == ';'
}

// xmlRefLen returns the length of the XML character reference at the start of s, or 0 if there
// isn't one. A "&" that opens no reference is ordinary text, so the ";" of "A &amp; B; C" ends a
// reference while the one in "A & B; C" separates attributes.
func xmlRefLen(s string) int {
	if len(s) == 0 || s[0] != '&' {
		return 0
	}
	end := strings.IndexByte(s, ';')
	if end < 2 {
		return 0
	}
	for i := 1; i < end; i++ {
		if !isXMLRefByte(s[i]) {
			return 0
		}
	}
	return end + 1
}

func isXMLRefByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '#' || c == '.' || c == '-' || c == '_' || c == ':'
}

// xmlRefASCII returns the ASCII character a reference stands for, or 0. Every separator is ASCII,
// and so is a quote, so anything wider is content whatever it decodes to. The named forms give 0
// as well: XML predefines only "&amp;", "&lt;", "&gt;", "&apos;" and "&quot;", none of which is DN
// syntax, and any other name would have failed the parse that got us here.
func xmlRefASCII(ref string) byte {
	digits, ok := strings.CutPrefix(ref[:len(ref)-1], "&#")
	if !ok {
		return 0
	}
	base := 10
	if hex, isHex := cutAnyPrefix(digits, "x", "X"); isHex {
		digits, base = hex, 16
	}
	code, err := strconv.ParseUint(digits, base, 32)
	if err != nil || code > unicode.MaxASCII {
		return 0
	}
	return byte(code)
}

func cutAnyPrefix(s string, prefixes ...string) (after string, found bool) {
	for _, prefix := range prefixes {
		if after, found = strings.CutPrefix(s, prefix); found {
			return after, true
		}
	}
	return s, false
}

// carriesSubstitution reports whether Fleet will replace something in this value. Host vitals count:
// like Fleet variables they are XML-escaped, but neither carries X.500 escaping.
func carriesSubstitution(value string) bool {
	return variables.Find(value) != nil || fleet.FindCustomHostVitalIDs(value) != nil
}

// substitutedQuote is how a quote inside a substituted value reaches the profile: every path that
// substitutes one escapes it with xml.EscapeText, which spells a quote this way and no other. So a
// quote in this spelling is content Fleet put there, while a raw quote or a "&quot;" is the admin's
// own X.500 writing, already saying what they meant it to say.
const substitutedQuote = "&#34;"

// doubleSubstitutedQuotes writes each substituted quote twice, which is how X.500 holds a quote
// inside a quoted value. Left single, the first one would end the value and turn the rest of it
// into attributes of the certificate's subject.
func doubleSubstitutedQuotes(value string) string {
	return strings.ReplaceAll(value, substitutedQuote, substitutedQuote+substitutedQuote)
}

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
