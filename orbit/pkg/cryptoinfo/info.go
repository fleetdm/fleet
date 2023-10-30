package cryptoinfo

import (
	"encoding/json"
)

type KeyInfo struct {
	Type     kiType
	Encoding kiEncoding
	Data     interface{}
	DataName kiDataNames
	Error    error
	Headers  map[string]string
}

// kiDataNames is an internal type. It's used to help provide uniformity in the returned data.
type kiDataNames string

const (
	kiCaCertificate kiDataNames = "certificate"
	kiCertificate               = "certificate"
	kiKey                       = "key"
)

// kiType is an internal type to denote what an indentified blob is. It is ultimately presented as a string
type kiType string

const (
	kiCACERTIFICATE kiType = "CA-CERTIFICATE" // Not totally sure what the correct string is here
	kiCERTIFICATE          = "CERTIFICATE"
	kiKEY                  = "KEY"
)

// kiType is an internal type to denote what encoding was used. It is ultimately presented as a string
type kiEncoding string

const (
	kiPEM kiEncoding = "PEM"
	kiDER            = "DER"
	kiP12            = "P12"
)

func NewKey(encoding kiEncoding) *KeyInfo {
	return &KeyInfo{
		DataName: kiKey,
		Encoding: encoding,
		Type:     kiKEY,
	}
}

func NewCertificate(encoding kiEncoding) *KeyInfo {
	return &KeyInfo{
		DataName: kiCertificate,
		Encoding: encoding,
		Type:     kiCERTIFICATE,
	}
}

func NewCaCertificate(encoding kiEncoding) *KeyInfo {
	return &KeyInfo{
		DataName: kiCaCertificate,
		Encoding: encoding,
		Type:     kiCACERTIFICATE,
	}
}

func NewError(encoding kiEncoding, err error) *KeyInfo {
	return &KeyInfo{
		Encoding: encoding,
		Error:    err,
	}
}

func (ki *KeyInfo) SetHeaders(headers map[string]string) *KeyInfo {
	ki.Headers = headers
	return ki
}

func (ki *KeyInfo) SetDataName(name kiDataNames) *KeyInfo {
	ki.DataName = name
	return ki
}

func (ki *KeyInfo) SetData(data interface{}, err error) *KeyInfo {
	ki.Data = data
	ki.Error = err
	return ki
}

// MarshalJSON is used by the go json marshaller. Using a custom one here
// allows us a high degree of control over the resulting output. For example,
// it allows us to use the same struct here to encapsulate both keys and
// certificate, and still have somewhat differenciated output
func (ki *KeyInfo) MarshalJSON() ([]byte, error) {
	// this feels somewhat inefficient WRT to allocations and shoving maps around. But it
	// also feels the simplest way to get consistent behavior without needing to push
	// the key/value pairs everywhere.
	ret := map[string]interface{}{
		"type":     ki.Type,
		"encoding": ki.Encoding,
	}

	if ki.Error != nil {
		ret["error"] = ki.Error.Error()
	} else {
		if ki.DataName != "" {
			ret[string(ki.DataName)] = ki.Data
		} else {
			ret["error"] = "No data name"
		}
	}

	if len(ki.Headers) != 0 {
		ret["headers"] = ki.Headers
	}

	return json.Marshal(ret)
}
