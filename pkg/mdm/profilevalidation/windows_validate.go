package profilevalidation

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/variables"
)

// windowsItem is one SyncML <Item> with a target. Data is nil when the item has no <Data>.
type windowsItem struct {
	LocURI     string  `xml:"Target>LocURI"`
	MetaFormat string  `xml:"Meta>Format"`
	Data       *string `xml:"Data"`
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
		var item windowsItem
		if err := dec.DecodeElement(&item, &el); err != nil {
			return nil, fmt.Errorf("decode profile: %w", err)
		}
		item.LocURI = strings.TrimSpace(item.LocURI)
		item.MetaFormat = strings.TrimSpace(item.MetaFormat)
		if item.LocURI != "" {
			items = append(items, item)
		}
	}
}

// windowsDataAccepts reports whether data parses as any of the node's formats. Only int, bool, b64,
// and xml are checked; every other format, including ones the validator doesn't know, accepts
// anything. Fleet variables need no special case: they expand to text, so they already fail int,
// bool, and b64 and pass chr and xml. A secret can hold any value, so it passes everywhere.
func windowsDataAccepts(formats WindowsFormats, data string) bool {
	data = strings.TrimSpace(data)
	if data == "" || len(formats) == 0 || variables.SecretRegex.MatchString(data) {
		return true
	}
	for _, f := range formats {
		switch f {
		case WindowsFormatInt:
			if _, err := strconv.ParseInt(data, 10, 64); err == nil {
				return true
			}
		case WindowsFormatBool:
			if slices.ContainsFunc([]string{"true", "false", "0", "1"}, func(b string) bool { return strings.EqualFold(b, data) }) {
				return true
			}
		case WindowsFormatB64:
			// Certificates are often line-wrapped.
			if _, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(data), "")); err == nil {
				return true
			}
		case WindowsFormatXML:
			if wellFormedXML(data) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func wellFormedXML(data string) bool {
	dec := xml.NewDecoder(strings.NewReader(data))
	for {
		_, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil {
			return false
		}
	}
}

// splitWindowsRoot splits a LocURI into its scope root and the rest of the path. root is
// WindowsScopeDevice or WindowsScopeUser for a ./Device/ or ./User/ path, and "" for a bare one.
func splitWindowsRoot(locURI string) (root WindowsScope, rest string) {
	rest, ok := strings.CutPrefix(locURI, "./")
	if !ok {
		return "", locURI
	}
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
	if !strings.HasPrefix(locURI, "./") {
		return nil, false
	}
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
