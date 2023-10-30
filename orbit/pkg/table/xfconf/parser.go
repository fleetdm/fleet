//go:build linux
// +build linux

package xfconf

import (
	"encoding/xml"
	"fmt"
	"os"
)

type (
	ArrayValue struct {
		Type  string `xml:"type,attr"`
		Value string `xml:"value,attr"`
	}
	Property struct {
		Name       string       `xml:"name,attr"`
		Type       string       `xml:"type,attr"`
		Value      string       `xml:"value,attr"`
		Properties []Property   `xml:"property"`
		Values     []ArrayValue `xml:"value"`
	}
	ChannelXML struct {
		XMLName     xml.Name   `xml:"channel"`
		ChannelName string     `xml:"name,attr"`
		Properties  []Property `xml:"property"`
	}
)

// parseXfconfXml reads in the given xml file, parses it as an xfconf XML file, and then flattens
// it. Because most XML elements in the xfconf files are `property`, we parse the file into
// a map with the name attributes set as the keys to avoid loss of meaningful full keys.
func parseXfconfXml(file string) (map[string]interface{}, error) {
	channelXml, err := readChannelXml(file)
	if err != nil {
		return nil, fmt.Errorf("could not read xfconf channel file %s: %w", file, err)
	}

	return channelXml.toMap(), nil
}

func readChannelXml(file string) (ChannelXML, error) {
	rdr, err := os.Open(file)
	if err != nil {
		return ChannelXML{}, err
	}

	xmlDecoder := xml.NewDecoder(rdr)

	var result ChannelXML
	xmlDecoder.Decode(&result)

	return result, nil
}

// toMap transforms Result r into a map where the top-level key is "channel/<name>".
func (c ChannelXML) toMap() map[string]interface{} {
	parentKey := fmt.Sprintf("channel/%s", c.ChannelName)

	properties := make(map[string]interface{}, 0)
	for _, p := range c.Properties {
		properties[p.Name] = p.mapValue()
	}

	results := make(map[string]interface{})
	results[parentKey] = properties

	return results
}

// mapValue transforms p into a value to be set in a map.
func (p Property) mapValue() interface{} {
	// This is an empty property with nested properties inside -- extract them.
	if len(p.Properties) > 0 {
		childPropertyMaps := make(map[string]interface{}, 0)
		for _, child := range p.Properties {
			// Call recursively for each child
			childPropertyMaps[child.Name] = child.mapValue()
		}

		return childPropertyMaps
	}

	// This property has a nested array -- extract the array.
	if p.Type == "array" {
		arrayValues := make([]interface{}, len(p.Values))
		for i, v := range p.Values {
			arrayValues[i] = v.Value
		}

		return arrayValues
	}

	// This property is a terminal/leaf property; just grab the value.
	return p.Value
}
