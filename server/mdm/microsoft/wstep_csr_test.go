package microsoft_mdm

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
)

func TestGetPublicKeyAlgorithmFromOID(t *testing.T) {
	testCases := []struct {
		oid      asn1.ObjectIdentifier
		expected x509.PublicKeyAlgorithm
	}{
		{oidPublicKeyRSA, x509.RSA},
		{oidPublicKeyDSA, x509.DSA},
		{oidPublicKeyECDSA, x509.ECDSA},
		{oidPublicKeyEd25519, x509.Ed25519},
		{asn1.ObjectIdentifier{0, 0}, x509.UnknownPublicKeyAlgorithm},
	}

	for _, tc := range testCases {
		t.Run(tc.oid.String(), func(t *testing.T) {
			result := getPublicKeyAlgorithmFromOID(tc.oid)
			require.Equal(t, tc.expected, result)
		})
	}
}

// The following tests were taken from the Go standard library (since the wstep
// code was taken from there as well)
// Copyright 2009 The Go Authors. All rights reserved.

var pemPrivateKey = testingKey(`
-----BEGIN RSA TESTING KEY-----
MIICXAIBAAKBgQCxoeCUW5KJxNPxMp+KmCxKLc1Zv9Ny+4CFqcUXVUYH69L3mQ7v
IWrJ9GBfcaA7BPQqUlWxWM+OCEQZH1EZNIuqRMNQVuIGCbz5UQ8w6tS0gcgdeGX7
J7jgCQ4RK3F/PuCM38QBLaHx988qG8NMc6VKErBjctCXFHQt14lerd5KpQIDAQAB
AoGAYrf6Hbk+mT5AI33k2Jt1kcweodBP7UkExkPxeuQzRVe0KVJw0EkcFhywKpr1
V5eLMrILWcJnpyHE5slWwtFHBG6a5fLaNtsBBtcAIfqTQ0Vfj5c6SzVaJv0Z5rOd
7gQF6isy3t3w9IF3We9wXQKzT6q5ypPGdm6fciKQ8RnzREkCQQDZwppKATqQ41/R
vhSj90fFifrGE6aVKC1hgSpxGQa4oIdsYYHwMzyhBmWW9Xv/R+fPyr8ZwPxp2c12
33QwOLPLAkEA0NNUb+z4ebVVHyvSwF5jhfJxigim+s49KuzJ1+A2RaSApGyBZiwS
rWvWkB471POAKUYt5ykIWVZ83zcceQiNTwJBAMJUFQZX5GDqWFc/zwGoKkeR49Yi
MTXIvf7Wmv6E++eFcnT461FlGAUHRV+bQQXGsItR/opIG7mGogIkVXa3E1MCQARX
AAA7eoZ9AEHflUeuLn9QJI/r0hyQQLEtrpwv6rDT1GCWaLII5HJ6NUFVf4TTcqxo
6vdM4QGKTJoO+SaCyP0CQFdpcxSAuzpFcKv0IlJ8XzS/cy+mweCMwyJ1PFEc4FX6
wg/HcAJWY60xZTJDFN+Qfx8ZQvBEin6c2/h+zZi5IVY=
-----END RSA TESTING KEY-----
`)

var testPrivateKey *rsa.PrivateKey

func init() {
	block, _ := pem.Decode([]byte(pemPrivateKey))

	var err error
	if testPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
		panic("Failed to parse private key: " + err.Error())
	}
}

func TestCreateCertificateRequest(t *testing.T) {
	random := rand.Reader

	ecdsa256Priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %s", err)
	}

	ecdsa384Priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %s", err)
	}

	ecdsa521Priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %s", err)
	}

	_, ed25519Priv, err := ed25519.GenerateKey(random)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %s", err)
	}

	tests := []struct {
		name    string
		priv    interface{}
		sigAlgo x509.SignatureAlgorithm
	}{
		{"RSA", testPrivateKey, x509.SHA1WithRSA},
		{"ECDSA-256", ecdsa256Priv, x509.ECDSAWithSHA1},
		{"ECDSA-384", ecdsa384Priv, x509.ECDSAWithSHA1},
		{"ECDSA-521", ecdsa521Priv, x509.ECDSAWithSHA1},
		{"Ed25519", ed25519Priv, x509.PureEd25519},
	}

	for _, test := range tests {
		template := x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   "test.example.com",
				Organization: []string{"Σ Acme Co"},
			},
			SignatureAlgorithm: test.sigAlgo,
			DNSNames:           []string{"test.example.com"},
			EmailAddresses:     []string{"gopher@golang.org"},
			IPAddresses:        []net.IP{net.IPv4(127, 0, 0, 1).To4(), net.ParseIP("2001:4860:0:2001::68")},
		}

		derBytes, err := x509.CreateCertificateRequest(random, &template, test.priv)
		if err != nil {
			t.Errorf("%s: failed to create certificate request: %s", test.name, err)
			continue
		}

		out, err := ParseCertificateRequestFromWindowsDevice(derBytes)
		if err != nil {
			t.Errorf("%s: failed to create certificate request: %s", test.name, err)
			continue
		}

		err = out.CheckSignature()
		if err != nil {
			t.Errorf("%s: failed to check certificate request signature: %s", test.name, err)
			continue
		}

		if out.Subject.CommonName != template.Subject.CommonName { //nolint:gocritic // ignore ifElseChain
			t.Errorf("%s: output subject common name and template subject common name don't match", test.name)
		} else if len(out.Subject.Organization) != len(template.Subject.Organization) {
			t.Errorf("%s: output subject organisation and template subject organisation don't match", test.name)
		} else if len(out.DNSNames) != len(template.DNSNames) {
			t.Errorf("%s: output DNS names and template DNS names don't match", test.name)
		} else if len(out.EmailAddresses) != len(template.EmailAddresses) {
			t.Errorf("%s: output email addresses and template email addresses don't match", test.name)
		} else if len(out.IPAddresses) != len(template.IPAddresses) {
			t.Errorf("%s: output IP addresses and template IP addresses names don't match", test.name)
		}
	}
}

func fromBase64(in string) []byte {
	out := make([]byte, base64.StdEncoding.DecodedLen(len(in)))
	n, err := base64.StdEncoding.Decode(out, []byte(in))
	if err != nil {
		panic("failed to base64 decode")
	}
	return out[:n]
}

func TestParseCertificateRequestFromWindowsDevice(t *testing.T) {
	for _, csrBase64 := range csrBase64Array {
		csrBytes := fromBase64(csrBase64)
		csr, err := ParseCertificateRequestFromWindowsDevice(csrBytes)
		if err != nil {
			t.Fatalf("failed to parse CSR: %s", err)
		}

		if len(csr.EmailAddresses) != 1 || csr.EmailAddresses[0] != "gopher@golang.org" {
			t.Errorf("incorrect email addresses found: %v", csr.EmailAddresses)
		}

		if len(csr.DNSNames) != 1 || csr.DNSNames[0] != "test.example.com" {
			t.Errorf("incorrect DNS names found: %v", csr.DNSNames)
		}

		if len(csr.Subject.Country) != 1 || csr.Subject.Country[0] != "AU" {
			t.Errorf("incorrect Subject name: %v", csr.Subject)
		}
	}
}

// These CSR was generated with OpenSSL:
//
//	openssl req -out CSR.csr -new -sha256 -nodes -keyout privateKey.key -config openssl.cnf
//
// With openssl.cnf containing the following sections:
//
//	[ v3_req ]
//	basicConstraints = CA:FALSE
//	keyUsage = nonRepudiation, digitalSignature, keyEncipherment
//	subjectAltName = email:gopher@golang.org,DNS:test.example.com
//	[ req_attributes ]
//	challengePassword = ignored challenge
//	unstructuredName  = ignored unstructured name
var csrBase64Array = [...]string{
	// Just [ v3_req ]
	"MIIDHDCCAgQCAQAwfjELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEUMBIGA1UEAwwLQ29tbW9uIE5hbWUxITAfBgkqhkiG9w0BCQEWEnRlc3RAZW1haWwuYWRkcmVzczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAK1GY4YFx2ujlZEOJxQVYmsjUnLsd5nFVnNpLE4cV+77sgv9NPNlB8uhn3MXt5leD34rm/2BisCHOifPucYlSrszo2beuKhvwn4+2FxDmWtBEMu/QA16L5IvoOfYZm/gJTsPwKDqvaR0tTU67a9OtxwNTBMI56YKtmwd/o8d3hYv9cg+9ZGAZ/gKONcg/OWYx/XRh6bd0g8DMbCikpWgXKDsvvK1Nk+VtkDO1JxuBaj4Lz/p/MifTfnHoqHxWOWl4EaTs4Ychxsv34/rSj1KD1tJqorIv5Xv2aqv4sjxfbrYzX4kvS5SC1goIovLnhj5UjmQ3Qy8u65eow/LLWw+YFcCAwEAAaBZMFcGCSqGSIb3DQEJDjFKMEgwCQYDVR0TBAIwADALBgNVHQ8EBAMCBeAwLgYDVR0RBCcwJYERZ29waGVyQGdvbGFuZy5vcmeCEHRlc3QuZXhhbXBsZS5jb20wDQYJKoZIhvcNAQELBQADggEBAB6VPMRrchvNW61Tokyq3ZvO6/NoGIbuwUn54q6l5VZW0Ep5Nq8juhegSSnaJ0jrovmUgKDN9vEo2KxuAtwG6udS6Ami3zP+hRd4k9Q8djJPb78nrjzWiindLK5Fps9U5mMoi1ER8ViveyAOTfnZt/jsKUaRsscY2FzE9t9/o5moE6LTcHUS4Ap1eheR+J72WOnQYn3cifYaemsA9MJuLko+kQ6xseqttbh9zjqd9fiCSh/LNkzos9c+mg2yMADitaZinAh+HZi50ooEbjaT3erNq9O6RqwJlgD00g6MQdoz9bTAryCUhCQfkIaepmQ7BxS0pqWNW3MMwfDwx/Snz6g=",
	// Both [ v3_req ] and [ req_attributes ]
	"MIIDaTCCAlECAQAwfjELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEUMBIGA1UEAwwLQ29tbW9uIE5hbWUxITAfBgkqhkiG9w0BCQEWEnRlc3RAZW1haWwuYWRkcmVzczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAK1GY4YFx2ujlZEOJxQVYmsjUnLsd5nFVnNpLE4cV+77sgv9NPNlB8uhn3MXt5leD34rm/2BisCHOifPucYlSrszo2beuKhvwn4+2FxDmWtBEMu/QA16L5IvoOfYZm/gJTsPwKDqvaR0tTU67a9OtxwNTBMI56YKtmwd/o8d3hYv9cg+9ZGAZ/gKONcg/OWYx/XRh6bd0g8DMbCikpWgXKDsvvK1Nk+VtkDO1JxuBaj4Lz/p/MifTfnHoqHxWOWl4EaTs4Ychxsv34/rSj1KD1tJqorIv5Xv2aqv4sjxfbrYzX4kvS5SC1goIovLnhj5UjmQ3Qy8u65eow/LLWw+YFcCAwEAAaCBpTAgBgkqhkiG9w0BCQcxEwwRaWdub3JlZCBjaGFsbGVuZ2UwKAYJKoZIhvcNAQkCMRsMGWlnbm9yZWQgdW5zdHJ1Y3R1cmVkIG5hbWUwVwYJKoZIhvcNAQkOMUowSDAJBgNVHRMEAjAAMAsGA1UdDwQEAwIF4DAuBgNVHREEJzAlgRFnb3BoZXJAZ29sYW5nLm9yZ4IQdGVzdC5leGFtcGxlLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEAgxe2N5O48EMsYE7o0rZBB0wi3Ov5/yYfnmmVI22Y3sP6VXbLDW0+UWIeSccOhzUCcZ/G4qcrfhhx6gTZTeA01nP7TdTJURvWAH5iFqj9sQ0qnLq6nEcVHij3sG6M5+BxAIVClQBk6lTCzgphc835Fjj6qSLuJ20XHdL5UfUbiJxx299CHgyBRL+hBUIPfz8p+ZgamyAuDLfnj54zzcRVyLlrmMLNPZNll1Q70RxoU6uWvLH8wB8vQe3Q/guSGubLyLRTUQVPh+dw1L4t8MKFWfX/48jwRM4gIRHFHPeAAE9D9YAoqdIvj/iFm/eQ++7DP8MDwOZWsXeB6jjwHuLmkQ==",
}

// TestDomainToReverseLabels covers the domain reverse-labels helper used by
// SAN URI validation. The function rejects domains with trailing dots or
// empty middle labels.
func TestDomainToReverseLabels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		domain string
		want   []string
		ok     bool
	}{
		{"three-label domain", "foo.example.com", []string{"com", "example", "foo"}, true},
		{"two-label domain", "example.com", []string{"com", "example"}, true},
		{"single label", "example", []string{"example"}, true},
		{"empty string yields no labels", "", nil, true},
		{"trailing dot is rejected", "foo.example.com.", nil, false},
		{"consecutive dots produce empty middle label", "foo..bar", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domainToReverseLabels(tc.domain)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestParseBase128Int pins the boundary and error cases of the base-128
// integer parser used inside parseTagAndLength. ASN.1 length-and-tag parsing
// bugs are a long-standing CVE class, so this function's guards deserve
// explicit coverage.
func TestParseBase128Int(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bytes      []byte
		offset     int
		wantRet    int
		wantOffset int
		errType    string // "" = no error, "syntax", "structural"
	}{
		{
			name:       "single-byte minimum",
			bytes:      []byte{0x05},
			wantRet:    5,
			wantOffset: 1,
		},
		{
			name:       "single-byte maximum without continuation",
			bytes:      []byte{0x7f},
			wantRet:    127,
			wantOffset: 1,
		},
		{
			name:       "multi-byte minimum above single-byte range",
			bytes:      []byte{0x81, 0x00},
			wantRet:    128,
			wantOffset: 2,
		},
		{
			name:    "non-minimal encoding rejected when leading byte is 0x80",
			bytes:   []byte{0x80, 0x01},
			errType: "syntax",
		},
		{
			name:    "truncated when continuation bit set but no more bytes",
			bytes:   []byte{0x81},
			errType: "syntax",
		},
		{
			name:    "rejected when integer exceeds five-byte limit",
			bytes:   []byte{0x81, 0x81, 0x81, 0x81, 0x81, 0x01},
			errType: "structural",
		},
		{
			name:    "rejected when value exceeds MaxInt32",
			bytes:   []byte{0xff, 0xff, 0xff, 0xff, 0x7f},
			errType: "structural",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ret, offset, err := parseBase128Int(tc.bytes, tc.offset)
			switch tc.errType {
			case "syntax":
				var se asn1.SyntaxError
				require.ErrorAs(t, err, &se, "expected asn1.SyntaxError")
			case "structural":
				var se asn1.StructuralError
				require.ErrorAs(t, err, &se, "expected asn1.StructuralError")
			default:
				require.NoError(t, err)
				require.Equal(t, tc.wantRet, ret)
				require.Equal(t, tc.wantOffset, offset)
			}
		})
	}
}

// TestParseTagAndLength pins the boundary and error cases of the ASN.1 tag
// and length parser. The DER-encoding rules around long-form length, minimal
// encoding, and indefinite-length rejection have all historically been
// sources of parser bugs.
func TestParseTagAndLength(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bytes      []byte
		offset     int
		wantTag    int
		wantClass  int
		wantLen    int
		wantOffset int
		errType    string
	}{
		{
			name:       "short tag and short length",
			bytes:      []byte{0x30, 0x05}, // SEQUENCE (tag 16), length 5
			wantTag:    0x10,
			wantClass:  0, // universal
			wantLen:    5,
			wantOffset: 2,
		},
		{
			name:       "long-form length encodes 256",
			bytes:      []byte{0x04, 0x82, 0x01, 0x00}, // OCTET STRING, length 256 in two bytes
			wantTag:    0x04,
			wantClass:  0,
			wantLen:    256,
			wantOffset: 4,
		},
		{
			name:    "indefinite length rejected",
			bytes:   []byte{0x30, 0x80}, // SEQUENCE with indefinite length marker
			errType: "syntax",
		},
		{
			name:    "non-minimal length rejected (short value in long form)",
			bytes:   []byte{0x04, 0x81, 0x05}, // long-form encoding of 5
			errType: "structural",
		},
		{
			name:    "length too large to shift safely",
			bytes:   []byte{0x04, 0x84, 0x80, 0x00, 0x00, 0x00}, // 4 bytes of length, overflows after shift
			errType: "structural",
		},
		{
			name:    "truncated length bytes",
			bytes:   []byte{0x04, 0x82, 0x01}, // long-form says 2 bytes follow, only 1 present
			errType: "syntax",
		},
		{
			name:    "non-minimal tag rejected",
			bytes:   []byte{0x1f, 0x05, 0x00}, // long-form tag header but value < 31
			errType: "syntax",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ret, offset, err := parseTagAndLength(tc.bytes, tc.offset)
			switch tc.errType {
			case "syntax":
				var se asn1.SyntaxError
				require.ErrorAs(t, err, &se, "expected asn1.SyntaxError")
			case "structural":
				var se asn1.StructuralError
				require.ErrorAs(t, err, &se, "expected asn1.StructuralError")
			default:
				require.NoError(t, err)
				require.Equal(t, tc.wantTag, ret.tag)
				require.Equal(t, tc.wantClass, ret.class)
				require.Equal(t, tc.wantLen, ret.length)
				require.Equal(t, tc.wantOffset, offset)
			}
		})
	}
}

// TestParseSANExtension covers Subject Alternative Name parsing for each
// name type that wstep_csr supports (DNS, email, URI, IP) plus the error
// path for malformed IP length. SAN parser bugs are a notable CVE class.
func TestParseSANExtension(t *testing.T) {
	// buildSAN encodes a single SAN extension SEQUENCE containing one entry
	// with the given context-specific tag and bytes.
	buildSAN := func(t *testing.T, contextTag byte, payload []byte) []byte {
		t.Helper()
		var b cryptobyte.Builder
		b.AddASN1(cryptobyte_asn1.SEQUENCE, func(b *cryptobyte.Builder) {
			b.AddASN1(cryptobyte_asn1.Tag(contextTag).ContextSpecific(), func(b *cryptobyte.Builder) {
				b.AddBytes(payload)
			})
		})
		out, err := b.Bytes()
		require.NoError(t, err)
		return out
	}

	t.Run("DNS name", func(t *testing.T) {
		der := buildSAN(t, nameTypeDNS, []byte("test.example.com"))
		dns, emails, ips, uris, err := parseSANExtension(der)
		require.NoError(t, err)
		require.Equal(t, []string{"test.example.com"}, dns)
		require.Empty(t, emails)
		require.Empty(t, ips)
		require.Empty(t, uris)
	})

	t.Run("email address", func(t *testing.T) {
		der := buildSAN(t, nameTypeEmail, []byte("gopher@golang.org"))
		dns, emails, ips, uris, err := parseSANExtension(der)
		require.NoError(t, err)
		require.Empty(t, dns)
		require.Equal(t, []string{"gopher@golang.org"}, emails)
		require.Empty(t, ips)
		require.Empty(t, uris)
	})

	t.Run("IPv4 address", func(t *testing.T) {
		ipv4 := net.IPv4(127, 0, 0, 1).To4()
		der := buildSAN(t, nameTypeIP, ipv4)
		dns, emails, ips, uris, err := parseSANExtension(der)
		require.NoError(t, err)
		require.Empty(t, dns)
		require.Empty(t, emails)
		require.Len(t, ips, 1)
		require.True(t, ips[0].Equal(ipv4))
		require.Empty(t, uris)
	})

	t.Run("IPv6 address", func(t *testing.T) {
		ipv6 := net.ParseIP("2001:db8::1")
		der := buildSAN(t, nameTypeIP, ipv6)
		dns, emails, ips, uris, err := parseSANExtension(der)
		require.NoError(t, err)
		require.Empty(t, dns)
		require.Empty(t, emails)
		require.Len(t, ips, 1)
		require.True(t, ips[0].Equal(ipv6))
		require.Empty(t, uris)
	})

	t.Run("URI", func(t *testing.T) {
		der := buildSAN(t, nameTypeURI, []byte("https://example.com/path"))
		dns, emails, ips, uris, err := parseSANExtension(der)
		require.NoError(t, err)
		require.Empty(t, dns)
		require.Empty(t, emails)
		require.Empty(t, ips)
		require.Len(t, uris, 1)
		require.Equal(t, "https://example.com/path", uris[0].String())
	})

	t.Run("invalid IP length is rejected", func(t *testing.T) {
		// 5 bytes is neither IPv4 (4) nor IPv6 (16).
		der := buildSAN(t, nameTypeIP, []byte{1, 2, 3, 4, 5})
		_, _, _, _, err := parseSANExtension(der)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot parse IP address")
	})

	t.Run("non-IA5 DNS name is rejected", func(t *testing.T) {
		// 0x80 is outside the ASCII range that IA5String allows.
		der := buildSAN(t, nameTypeDNS, []byte{'a', 0x80, 'b'})
		_, _, _, _, err := parseSANExtension(der)
		require.Error(t, err)
		require.Contains(t, err.Error(), "malformed")
	})
}
