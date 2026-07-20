package sigverify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const osslsigncodeOKOutput = `Current PE checksum   : 0009D06A
Calculated PE checksum: 0009D06A

Message digest algorithm  : SHA256
Current message digest    : 1A2B3C4D
Calculated message digest : 1A2B3C4D

Signature Index: 0  (Primary Signature)

Signer's certificate:
	Signer #0:
		Subject: /C=GB/O=Simon Tatham/CN=Simon Tatham
		Issuer : /C=GB/O=Sectigo Limited/CN=Sectigo Public Code Signing CA R36
		Serial : 00ABCDEF
		Certificate expiration date:
			notBefore : Jan  1 00:00:00 2025 GMT
			notAfter : Jan  1 23:59:59 2028 GMT

Number of certificates: 4
	Cert #0:
		Subject: /C=GB/O=Sectigo Limited/CN=Sectigo Public Code Signing Root R46
		Issuer : /C=US/ST=New Jersey/O=The USERTRUST Network/CN=USERTrust RSA Certification Authority
	Cert #1:
		Subject: /C=GB/O=Sectigo Limited/CN=Sectigo Public Code Signing CA R36
		Issuer : /C=GB/O=Sectigo Limited/CN=Sectigo Public Code Signing Root R46
	Cert #2:
		Subject: /C=GB/O=Simon Tatham/CN=Simon Tatham
		Issuer : /C=GB/O=Sectigo Limited/CN=Sectigo Public Code Signing CA R36

Authenticated attributes:
	Message digest algorithm: SHA256

Timestamp Server Signature verification: ok
Signature verification: ok

Number of verified signatures: 1
Succeeded
`

// osslsigncode 2.13 output shape: RFC 2253 comma-separated DNs, and TSA/CRL
// verification sections that repeat "Signer #N:"/"Subject:" lines for
// certificates that are NOT the signer.
const osslsigncode213Output = `Warning: MsiDigitalSignatureEx stream doesn't exist

Signature Index: 0  (Primary Signature)

Message digest algorithm         : SHA1
Current DigitalSignature         : 8A17866CD7B2DFA570677E222AAF65B6D9B041DF
Calculated DigitalSignature      : 8A17866CD7B2DFA570677E222AAF65B6D9B041DF

Signer's certificate:
	------------------
	Signer #0:
		Subject: CN=Simon Tatham,O=Simon Tatham,ST=Cambridgeshire,C=GB
		Issuer : CN=Sectigo Public Code Signing CA R36,O=Sectigo Limited,C=GB
		Serial : BE8E1D85C5D2521B6D33379E3B8501A9
		Certificate expiration date:
			notBefore : Sep 27 00:00:00 2024 GMT
			notAfter : Sep 27 23:59:59 2027 GMT

Message digest algorithm: SHA1

Authenticated attributes:
	Signing time: May 14 19:35:07 2026 GMT
	Microsoft Individual Code Signing purpose

Countersignatures:
	Timestamp time: May 14 19:35:10 2026 GMT

CAfile: /etc/ssl/cert.pem
TSA's certificates file: /etc/ssl/cert.pem

Timestamp verified using:
	------------------
	Signer #1:
		Subject: CN=Sectigo Public Time Stamping CA R36,O=Sectigo Limited,C=GB
		Issuer : CN=Sectigo Public Time Stamping Root R46,O=Sectigo Limited,C=GB
	------------------
	Signer #0:
		Subject: CN=Sectigo Public Time Stamping Signer R36,O=Sectigo Limited,ST=West Yorkshire,C=GB
		Issuer : CN=Sectigo Public Time Stamping CA R36,O=Sectigo Limited,C=GB

Certificate Revocation List verified using:
	------------------
	Signer #1:
		Subject: CN=Sectigo Public Code Signing CA R36,O=Sectigo Limited,C=GB
		Issuer : CN=Sectigo Public Code Signing Root R46,O=Sectigo Limited,C=GB
	------------------
	Signer #0:
		Subject: CN=Simon Tatham,O=Simon Tatham,ST=Cambridgeshire,C=GB
		Issuer : CN=Sectigo Public Code Signing CA R36,O=Sectigo Limited,C=GB

Signature CRL verification: ok
Signature verification: ok

Number of verified signatures: 2
Succeeded
`

const osslsigncodeNoSignatureMSI = `MSI file has no signature.
Failed
`

const osslsigncodeNoSignaturePE = `No signature found

Failed
`

const osslsigncodeFailedOutput = `Signature Index: 0  (Primary Signature)

Signer's certificate:
	Signer #0:
		Subject: /C=US/O=Evil Corp/CN=Evil Corp
		Issuer : /C=US/O=Evil Corp/CN=Evil Corp Root

Signature verification: failed

Number of verified signatures: 1
Failed
`

func TestParseOsslsigncodeOutput(t *testing.T) {
	t.Run("verified signature", func(t *testing.T) {
		res := ParseOsslsigncodeOutput(osslsigncodeOKOutput)
		require.True(t, res.Verified)
		require.False(t, res.NoSignature)
		// Only the signer's CN, not the chain CAs from "Number of certificates".
		require.Equal(t, []string{"Simon Tatham"}, res.SubjectCNs)
	})

	t.Run("verified signature, osslsigncode 2.13 format", func(t *testing.T) {
		res := ParseOsslsigncodeOutput(osslsigncode213Output)
		require.True(t, res.Verified)
		require.False(t, res.NoSignature)
		// Only the leaf signer's CN — not the TSA or CRL chain certificates
		// from the later "Signer #N:" sections.
		require.Equal(t, []string{"Simon Tatham"}, res.SubjectCNs)
	})

	t.Run("unsigned MSI", func(t *testing.T) {
		res := ParseOsslsigncodeOutput(osslsigncodeNoSignatureMSI)
		require.False(t, res.Verified)
		require.True(t, res.NoSignature)
	})

	t.Run("unsigned PE", func(t *testing.T) {
		res := ParseOsslsigncodeOutput(osslsigncodeNoSignaturePE)
		require.False(t, res.Verified)
		require.True(t, res.NoSignature)
	})

	t.Run("failed verification", func(t *testing.T) {
		res := ParseOsslsigncodeOutput(osslsigncodeFailedOutput)
		require.False(t, res.Verified)
		require.False(t, res.NoSignature)
		require.Equal(t, []string{"Evil Corp"}, res.SubjectCNs)
		require.Contains(t, res.Detail, "Signature verification: failed")
	})
}

func TestSubjectCNFromDN(t *testing.T) {
	testCases := []struct {
		dn   string
		want string
	}{
		{"/C=US/ST=California/L=Redwood City/O=Box, Inc./CN=Box, Inc.", "Box, Inc."},
		{"/C=GB/O=Simon Tatham/CN=Simon Tatham", "Simon Tatham"},
		{"/CN=Bandisoft International Inc./O=Bandisoft", "Bandisoft International Inc."},
		{"/O=NoCommonName Corp", ""},
		{"/C=US/CN=Name With /slash inside", "Name With /slash inside"},
		// RFC 2253 comma-separated format (osslsigncode 2.x)
		{"CN=Simon Tatham,O=Simon Tatham,ST=Cambridgeshire,C=GB", "Simon Tatham"},
		{`CN=Box\, Inc.,O=Box\, Inc.,L=Redwood City,ST=California,C=US`, "Box, Inc."},
		{"O=NoCN Corp,C=US", ""},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.want, SubjectCNFromDN(tc.dn), "dn: %s", tc.dn)
	}
}

func TestSubjectCNFromX500DN(t *testing.T) {
	testCases := []struct {
		dn   string
		want string
	}{
		{`CN="Box, Inc.", O="Box, Inc.", L=Redwood City, S=California, C=US`, "Box, Inc."},
		{"CN=Simon Tatham, O=Simon Tatham, S=Cambridgeshire, C=GB", "Simon Tatham"},
		{"O=No CN Corp, C=US", ""},
		{"", ""},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.want, SubjectCNFromX500DN(tc.dn), "dn: %s", tc.dn)
	}
}
