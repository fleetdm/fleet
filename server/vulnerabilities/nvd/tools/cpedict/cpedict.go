// Package cpedict defines the types and methods necessary to parse and lookup CPE dictionary conforming to
// CPE Dictionary specification 2.3 as per https://nvlpubs.nist.gov/nistpubs/Legacy/IR/nistir7697.pdf.
// The implementation is not full, only parts required to parse NVD vulnerability feed are implemented
//
// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cpedict

import (
	"encoding/xml"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// TextType represents multi-language text
type TextType map[string]string

// UnmarshalXML -- load TextType from XML
func (t *TextType) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var text string
	lang := "en"
	if *t == nil {
		*t = TextType{}
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "lang" {
			lang = attr.Value
		}
	}
	if err := d.DecodeElement(&text, &start); err != nil {
		return err
	}
	(*t)[lang] = text
	return nil
}

// PlatformType -- NVD doesn't use it
// TODO: implement
// type PlatformType struct{}

// CheckFactRefType is a reference to a check that always evaluates to
// TRUE, FALSE, or ERROR. Examples of types of checks are OVAL and OCIL checks.
// NVD doesn't use it
// TODO: implement
// type CheckFactRefType struct{}

// NamePattern represents CPE name
type NamePattern wfn.Attributes

// UnmarshalXMLAttr implements xml.UnmarshalerAttr interface
func (np *NamePattern) UnmarshalXMLAttr(attr xml.Attr) error {
	wfn, err := wfn.Parse(attr.Value)
	if err != nil {
		return err
	}
	*np = (NamePattern)(*wfn)
	return nil
}

func (np NamePattern) String() string {
	return wfn.Attributes(np).String()
}

// Reference holds additional information about CPE.
type Reference struct {
	URL  string `xml:"href,attr"`
	Desc string `xml:",chardata"`
}

// DeprecatedInfo contains the name that is deprecating the identifier name and the type of Deprecation
type DeprecatedInfo struct {
	Name NamePattern `xml:"name,attr"`
	Type string      `xml:"type,attr"`
}

// Deprecation contains the deprecation information for a specific deprecation of a given identifier name.
type Deprecation struct {
	Date         time.Time        `xml:"date,attr"`
	DeprecatedBy []DeprecatedInfo `xml:"deprecated-by"`
}

// CPE23Item contains all CPE 2.3 specific data related to a given identifier name.
type CPE23Item struct {
	Name        NamePattern  `xml:"name,attr"`
	Deprecation *Deprecation `xml:"deprecation"`
	// TODO: implement ProvenanceRecord
}

// CPEItem contains all of the information for a single dictionary entry (identifier name), including metadata.
type CPEItem struct {
	Name            NamePattern  `xml:"name,attr"`
	Deprecated      bool         `xml:"deprecated,attr"`
	DeprecatedBy    *NamePattern `xml:"deprecated_by,attr"`
	DeprecationDate time.Time    `xml:"deprecation_date,attr"`
	CPE23           CPE23Item    `xml:"cpe23-item"`
	Title           TextType     `xml:"title"`
	Notes           TextType     `xml:"notes"`
	References      []Reference  `xml:"references>reference"`
	// Calls out a check, such as an OVAL definition, that can confirm or reject
	// an IT system as an instance of the named platform. 0-n occurrences.
	// TODO: not implemented
	Check struct{} `xml:"check"`
}

// Generator contains information about the generation of the dictionary file.
type Generator struct {
	ProductName    string    `xml:"product_name"`
	ProductVersion string    `xml:"product_version"`
	SchemaVersion  string    `xml:"schema_version"`
	TimeStamp      time.Time `xml:"timestamp"`
}

// CPEList  contains all of the dictionary entries and dictionary metadata.
type CPEList struct {
	Generator Generator `xml:"generator"`
	Items     []CPEItem `xml:"cpe-item"`
}

// Decode decodes dictionary XML
func Decode(r io.Reader) (*CPEList, error) {
	var list CPEList
	if err := xml.NewDecoder(r).Decode(&list); err != nil {
		return nil, err
	}
	return &list, nil
}
