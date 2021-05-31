package sso

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	dsigtypes "github.com/russellhaering/goxmldsig/types"
)

type Metadata struct {
	XMLName          xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID         string           `xml:"entityID,attr"`
	IDPSSODescriptor IDPSSODescriptor `xml:"IDPSSODescriptor"`
}

type IDPSSODescriptor struct {
	XMLName             xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	KeyDescriptors      []KeyDescriptor       `xml:"KeyDescriptor"`
	NameIDFormats       []NameIDFormat        `xml:"NameIDFormat"`
	SingleSignOnService []SingleSignOnService `xml:"SingleSignOnService"`
	Attributes          []Attribute           `xml:"Attribute"`
}

type KeyDescriptor struct {
	XMLName xml.Name          `xml:"urn:oasis:names:tc:SAML:2.0:metadata KeyDescriptor"`
	Use     string            `xml:"use,attr"`
	KeyInfo dsigtypes.KeyInfo `xml:"KeyInfo"`
}

type NameIDFormat struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata NameIDFormat"`
	Value   string   `xml:",chardata"`
}

type SingleSignOnService struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata SingleSignOnService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
}

const (
	PasswordProtectedTransport = "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport"
	RedirectBinding            = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
)

type Settings struct {
	Metadata *Metadata
	// AssertionConsumerServiceURL is the call back on the service provider which responds
	// to the IDP
	AssertionConsumerServiceURL string
	SessionStore                SessionStore
	OriginalURL                 string
}

// ParseMetadata writes metadata xml to a struct
func ParseMetadata(metadata string) (*Metadata, error) {
	var md Metadata
	err := xml.Unmarshal([]byte(metadata), &md)
	if err != nil {
		return nil, err
	}
	return &md, nil
}

// GetMetadata retrieves information describing how to interact with a particular
// IDP via a remote URL. metadataURL is the location where the metadata is located
// and timeout defines how long to wait to get a response form the metadata
// server.
func GetMetadata(metadataURL string) (*Metadata, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("SAML metadata server at %s returned %s", metadataURL, resp.Status)
	}
	xmlData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}
	var md Metadata
	err = xml.Unmarshal(xmlData, &md)
	if err != nil {
		return nil, err
	}
	return &md, nil
}
