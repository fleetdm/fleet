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
