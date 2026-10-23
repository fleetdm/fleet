package profilevalidation

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
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
