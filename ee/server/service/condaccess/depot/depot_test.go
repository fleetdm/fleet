package depot

import (
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestExtractUUIDFromCert_AppleFormat(t *testing.T) {
	const wantUUID = "AABBCCDD-1122-3344-5566-778899AABBCC"
	crt := &x509.Certificate{
		URIs: []*url.URL{mustParseURL("urn:device:apple:uuid:" + wantUUID)},
	}
	assert.Equal(t, wantUUID, extractUUIDFromCert(crt))
}

func TestExtractUUIDFromCert_FleetFormat(t *testing.T) {
	const wantUUID = "DEADBEEF-CAFE-BABE-F00D-BAADDEADBEEF"
	crt := &x509.Certificate{
		URIs: []*url.URL{mustParseURL("urn:device:fleet:uuid:" + wantUUID)},
	}
	assert.Equal(t, wantUUID, extractUUIDFromCert(crt))
}

func TestExtractUUIDFromCert_NoURI(t *testing.T) {
	crt := &x509.Certificate{}
	assert.Equal(t, "", extractUUIDFromCert(crt))
}

func TestExtractUUIDFromCert_UnknownPrefix(t *testing.T) {
	crt := &x509.Certificate{
		URIs: []*url.URL{mustParseURL("urn:device:windows:uuid:SOME-UUID")},
	}
	assert.Equal(t, "", extractUUIDFromCert(crt))
}
