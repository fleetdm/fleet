package profilevalidation

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/variables"
)

// windowsItem is one SyncML <Item> with a target. Data is the text of <Data> (CDATA unwrapped,
// entities decoded), or nil when the item has none. DataXML is its raw content when <Data> has
// child elements, which the text leaves out.
type windowsItem struct {
	LocURI     string
	MetaFormat string
	Data       *string
	DataXML    string
}

// decodeWindowsProfile returns every targeted <Item> in a SyncML profile, wherever it's nested
// (Replace, Add, Exec, Atomic). It doesn't import server/fleet's SyncML types: that package will
// call this validator.
func decodeWindowsProfile(data []byte) ([]windowsItem, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var items []windowsItem
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("decode profile: %w", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Local != "Item" {
			continue
		}
		var raw struct {
			LocURI     string `xml:"Target>LocURI"`
			MetaFormat string `xml:"Meta>Format"`
			Data       *struct {
				Text     string     `xml:",chardata"`
				Inner    string     `xml:",innerxml"`
				Children []struct{} `xml:",any"`
			} `xml:"Data"`
		}
		if err := dec.DecodeElement(&raw, &el); err != nil {
			return nil, fmt.Errorf("decode profile: %w", err)
		}
		item := windowsItem{LocURI: strings.TrimSpace(raw.LocURI), MetaFormat: strings.TrimSpace(raw.MetaFormat)}
		if raw.Data != nil {
			item.Data = &raw.Data.Text
			if len(raw.Data.Children) > 0 {
				item.DataXML = raw.Data.Inner
			}
		}
		if item.LocURI != "" {
			items = append(items, item)
		}
	}
}

// windowsDataAccepts reports whether an item's Data parses as any of the node's formats. dataXML
// is the raw content when <Data> has child elements. Only int, bool, b64, and xml are checked, and
// an empty value fails them (Windows rejects it); every other format, including ones the validator
// doesn't know, accepts anything. Fleet variables need no special case: they expand to text, so
// they already fail int, bool, and b64. A secret can hold any value, so it passes everywhere.
func windowsDataAccepts(formats WindowsFormats, data, dataXML string) bool {
	data = strings.TrimSpace(data)
	if len(formats) == 0 || variables.SecretRegex.MatchString(data) || variables.SecretRegex.MatchString(dataXML) {
		return true
	}
	// Child elements make a scalar invalid even if the text around them parses.
	scalar := data
	if dataXML != "" {
		scalar = ""
	}
	for _, f := range formats {
		switch f {
		case WindowsFormatInt:
			if _, err := strconv.ParseInt(scalar, 10, 64); err == nil {
				return true
			}
		case WindowsFormatBool:
			if slices.ContainsFunc([]string{"true", "false", "0", "1"}, func(b string) bool { return strings.EqualFold(b, scalar) }) {
				return true
			}
		case WindowsFormatB64:
			// Certificates are often line-wrapped.
			b64 := strings.Join(strings.Fields(scalar), "")
			if _, err := base64.StdEncoding.DecodeString(b64); b64 != "" && err == nil {
				return true
			}
		case WindowsFormatXML:
			// Raw child elements are the document; otherwise it's escaped or in CDATA.
			doc := data
			if dataXML != "" {
				doc = dataXML
			}
			if wellFormedXML(doc) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// wellFormedXML reports whether data is one XML document: a single root element, with only
// whitespace, comments, and processing instructions around it.
func wellFormedXML(data string) bool {
	dec := xml.NewDecoder(strings.NewReader(data))
	depth, roots := 0, 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return roots == 1
		}
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) > 0 {
				return false
			}
		}
	}
}

// splitWindowsRoot splits a LocURI into its scope root and the rest of the path. root is
// WindowsScopeDevice or WindowsScopeUser for a ./Device/ or ./User/ path, and "" for a bare one.
// The leading "./" is optional, as on devices.
func splitWindowsRoot(locURI string) (root WindowsScope, rest string) {
	rest = strings.TrimPrefix(locURI, "./")
	first, after, _ := strings.Cut(rest, "/")
	for _, scope := range []WindowsScope{WindowsScopeDevice, WindowsScopeUser} {
		if strings.EqualFold(first, string(scope)) {
			return scope, after
		}
	}
	return "", rest
}

// resolveWindowsNode finds the node a LocURI targets. A bare ./Vendor/MSFT/X and
// ./Device/Vendor/MSFT/X are the same setting, so each is tried for the other. If neither spelling
// matches, the other scope is tried and wrongScope is set; a bare path counts as device scope.
func (s *ProfileValidationSchema) resolveWindowsNode(locURI string) (node *WindowsNode, wrongScope bool) {
	root, rest := splitWindowsRoot(locURI)
	device, bare, user := "./Device/"+rest, "./"+rest, "./User/"+rest
	same, other := []string{device, bare}, []string{user}
	switch root {
	case "":
		same = []string{bare, device}
	case WindowsScopeUser:
		same, other = []string{user}, []string{device, bare}
	}
	// The DDF declares Policy/Config/{Area}/{Policy} as dynamic nodes so ADMX ingestion can add
	// areas. Matching them would make every policy LocURI known, so policies must match exactly:
	// a DDF policy or a template policy.
	var catchAll []*WindowsNode
	for _, r := range []string{"./Device", "./User"} {
		for _, p := range []string{"/Vendor/MSFT/Policy/Config/*", "/Vendor/MSFT/Policy/Config/*/*"} {
			if n := s.Windows[r+p]; n != nil {
				catchAll = append(catchAll, n)
			}
		}
	}
	lookup := func(paths []string) *WindowsNode {
		for _, p := range paths {
			if n := s.WindowsNode(p); n != nil && !slices.Contains(catchAll, n) {
				return n
			}
			if n := s.windowsTemplateNode(p); n != nil {
				return n
			}
		}
		return nil
	}
	if n := lookup(same); n != nil {
		return n, false
	}
	if n := lookup(other); n != nil {
		return n, true
	}
	return nil, false
}

// windowsTemplateNode matches an ingested ADMX policy ({App}~Policy~{Category}/{Policy}) to a
// template policy regardless of {App}: the admin picks it in ADMXInstall, so a renamed install of
// a template we know is still that template.
func (s *ProfileValidationSchema) windowsTemplateNode(locURI string) *WindowsNode {
	prefix, suffix, ok := cutWindowsADMXApp(locURI)
	if !ok {
		return nil
	}
	var best *WindowsNode
	bestKey := ""
	// ponytail: scans every key; index template keys by suffix at Parse if lookups get hot.
	for key, n := range s.Windows {
		if n == nil || n.Source != WindowsSourceADMXTemplate {
			continue
		}
		kp, ks, ok := cutWindowsADMXApp(key)
		// Ties break on the key so the result doesn't depend on map order.
		if ok && strings.EqualFold(kp, prefix) && strings.EqualFold(ks, suffix) && (best == nil || key < bestKey) {
			best, bestKey = n, key
		}
	}
	return best
}

// cutWindowsADMXApp splits a LocURI around the app name of its ~Policy~ area, e.g.
// "./Device/Vendor/MSFT/Policy/Config/", "~Policy~googlechrome/HomepageLocation".
func cutWindowsADMXApp(locURI string) (prefix, suffix string, ok bool) {
	// Not strings.ToLower: it can change byte lengths (Ⱥ → ⱥ), so its index wouldn't fit locURI.
	loc := windowsADMXMarkerRegex.FindStringIndex(locURI)
	if loc == nil {
		return "", "", false
	}
	i := loc[0]
	return locURI[:strings.LastIndex(locURI[:i], "/")+1], locURI[i:], true
}

var windowsADMXMarkerRegex = regexp.MustCompile(`(?i)~policy~`)

// windowsArea names what a LocURI configures for findings: the area for a Policy CSP setting,
// e.g. DeviceLock, and otherwise the CSP, e.g. Firewall or DMAcc.
func windowsArea(locURI string) string {
	_, rest := splitWindowsRoot(locURI)
	segs := strings.Split(rest, "/")
	switch {
	case len(segs) >= 2 && strings.EqualFold(segs[0], "Vendor") && strings.EqualFold(segs[1], "MSFT"):
		segs = segs[2:]
	case len(segs) >= 1 && strings.EqualFold(segs[0], "SyncML"):
		segs = segs[1:]
	}
	if len(segs) >= 3 && strings.EqualFold(segs[0], "Policy") && strings.EqualFold(segs[1], "Config") {
		return segs[2]
	}
	if len(segs) == 0 {
		return ""
	}
	return segs[0]
}

var knownWindowsFormats = []WindowsFormat{
	WindowsFormatInt, WindowsFormatChr, WindowsFormatBool, WindowsFormatB64, WindowsFormatXML,
	WindowsFormatNode, WindowsFormatNull, WindowsFormatBin, WindowsFormatTime, WindowsFormatDate,
	WindowsFormatFloat,
}

// ValidateWindowsProfile checks a SyncML profile's items against the schema. It runs after the
// structural validator, so it only reports schema problems; malformed XML returns an error.
// Lenient mode reports Data that doesn't parse as the node's format. Strict mode also reports
// unknown LocURIs, a <Meta><Format> that disagrees with the DDF, and a path in the wrong scope.
func (s *ProfileValidationSchema) ValidateWindowsProfile(data []byte, mode Mode) ([]Finding, error) {
	items, err := decodeWindowsProfile(data)
	if err != nil {
		return nil, err
	}
	strict := mode == ModeStrict
	var findings []Finding
	for _, item := range items {
		finding := func(code Code, expected []string, actual string) {
			findings = append(findings, Finding{
				Code: code, PayloadType: windowsArea(item.LocURI), KeyPath: item.LocURI,
				Expected: expected, Actual: actual,
			})
		}
		node, wrongScope := s.resolveWindowsNode(item.LocURI)
		if node == nil {
			if strict {
				finding(CodeUnknownKey, nil, "")
			}
			continue
		}
		formats := make([]string, 0, len(node.Format))
		known := true
		for _, f := range node.Format {
			formats = append(formats, string(f))
			known = known && slices.Contains(knownWindowsFormats, f)
		}
		if strict && wrongScope {
			scope, _ := splitWindowsRoot(item.LocURI)
			if scope == "" {
				scope = WindowsScopeDevice
			}
			finding(CodeWrongScope, []string{string(node.Scope)}, string(scope))
		}
		if strict && known && item.MetaFormat != "" &&
			!slices.ContainsFunc(formats, func(f string) bool { return strings.EqualFold(f, item.MetaFormat) }) {
			finding(CodeFormatMismatch, formats, item.MetaFormat)
		}
		// Actual is left empty: echoing Data could leak values such as SCEP challenges.
		if item.Data != nil && !windowsDataAccepts(node.Format, *item.Data, item.DataXML) {
			finding(CodeTypeMismatch, formats, "")
		}
	}
	return findings, nil
}
