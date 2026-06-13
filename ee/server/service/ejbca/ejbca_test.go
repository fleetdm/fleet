package ejbca

import (
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildUPNSANExtension verifies the otherName-based subjectAltName encoding
// roundtrips through Go's encoding/asn1 — a focused sanity check for the only
// hand-rolled ASN.1 in this package. A failing assert here means EJBCA would
// reject the CSR or strip the UPN; far cheaper to catch in a unit test than
// during a real enrollment.
func TestBuildUPNSANExtension(t *testing.T) {
	upns := []string{"alice@corp.example.com", "alice.alt@corp.example.com"}

	ext, err := buildUPNSANExtension(upns)
	require.NoError(t, err)
	assert.Equal(t, oidSubjectAltName, ext.Id)
	assert.NotEmpty(t, ext.Value)

	// Parse SubjectAltName ::= SEQUENCE OF GeneralName.
	var generalNames []asn1.RawValue
	rest, err := asn1.Unmarshal(ext.Value, &generalNames)
	require.NoError(t, err)
	assert.Empty(t, rest)
	require.Len(t, generalNames, len(upns))

	type upnValue struct {
		Value string `asn1:"utf8"`
	}
	type otherName struct {
		TypeID asn1.ObjectIdentifier
		Value  upnValue `asn1:"explicit,tag:0"`
	}

	for i, gn := range generalNames {
		// Each GeneralName.otherName has the context-specific [0] (0xA0) tag.
		// Rewrite the outer byte back to universal SEQUENCE (0x30) so the
		// standard unmarshaler decodes it.
		fullBytes := append([]byte(nil), gn.FullBytes...)
		require.Equal(t, byte(0xA0), fullBytes[0], "otherName outer tag should be context-specific [0] constructed")
		fullBytes[0] = 0x30

		var on otherName
		_, err := asn1.Unmarshal(fullBytes, &on)
		require.NoError(t, err, "unmarshal otherName %d", i)
		assert.True(t, on.TypeID.Equal(oidMicrosoftUPN), "OID mismatch on otherName %d", i)
		assert.Equal(t, upns[i], on.Value.Value, "UPN value mismatch on otherName %d", i)
	}
}

// TestBuildUPNSANExtension_Empty confirms the function handles an empty input
// slice gracefully. The caller in GetCertificate guards against this anyway,
// but defensive behavior is cheap to verify.
func TestBuildUPNSANExtension_Empty(t *testing.T) {
	ext, err := buildUPNSANExtension(nil)
	require.NoError(t, err)
	assert.Equal(t, oidSubjectAltName, ext.Id)
	// An empty SEQUENCE OF — 0x30 0x00 — is well-formed ASN.1.
	assert.Equal(t, []byte{0x30, 0x00}, ext.Value)
}
