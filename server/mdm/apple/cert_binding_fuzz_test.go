package apple_mdm

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func FuzzParseAppleMDMCertificateBindingExtension(f *testing.F) {
	for _, raw := range []string{`{"v":1,"purpose":"ade"}`, `{"v":2}`, `{`, `{"v":1} {}`, `null`} {
		value, err := asn1.MarshalWithParams(raw, "utf8")
		require.NoError(f, err)
		f.Add(value, "synthetic-id", uint8(0))
	}
	f.Add([]byte{0x0c, 0x80}, "", uint8(3))
	f.Fuzz(func(t *testing.T, raw []byte, identifier string, mode uint8) {
		if len(raw) > 4096 || len(identifier) > 512 {
			t.Skip()
		}
		extension := pkix.Extension{Id: AppleMDMCertificateBindingExtensionOID, Value: raw}
		cert := &x509.Certificate{Extensions: []pkix.Extension{extension}}
		parsed, err := ParseAppleMDMCertificateBindingExtension(cert)
		if err == nil {
			// A successful parse must consume exactly one ASN.1 UTF8String
			// containing one JSON value with the supported schema version.
			var text string
			rest, decodeErr := asn1.UnmarshalWithParams(raw, &text, "utf8")
			require.NoError(t, decodeErr)
			require.Empty(t, rest)
			var expected AppleMDMCertificateBindingExtension
			require.NoError(t, json.Unmarshal([]byte(text), &expected))
			require.Equal(t, 1, expected.Version)
			require.Equal(t, &expected, parsed)
		}
		cert.Extensions = append(cert.Extensions, extension)
		_, err = ParseAppleMDMCertificateBindingExtension(cert)
		require.Error(t, err)
		absent, err := ParseAppleMDMCertificateBindingExtension(&x509.Certificate{})
		require.NoError(t, err)
		require.Nil(t, absent)

		// JSON normalizes invalid UTF-8. Normalize before comparing the
		// public builder/parser round trip, while leaving raw input arbitrary.
		identifier = strings.ToValidUTF8(identifier, "\ufffd")
		expected := AppleMDMCertificateBindingExtension{Version: 1, Purpose: fleet.AppleMDMCertPurpose("ade")}
		switch mode % 4 {
		case 0:
			expected.UDID, expected.Serial = &identifier, &identifier
		case 1:
			expected.IDPAccountUUID = &identifier
		case 2:
			expected.EnrollmentID = &identifier
		}
		built, err := BuildAppleMDMCertificateBindingExtension(expected)
		require.NoError(t, err)
		cert.Extensions = []pkix.Extension{built}
		parsed, err = ParseAppleMDMCertificateBindingExtension(cert)
		require.NoError(t, err)
		require.Equal(t, &expected, parsed)
		cert.Extensions[0].Value = append(append([]byte(nil), built.Value...), 0)
		_, err = ParseAppleMDMCertificateBindingExtension(cert)
		require.Error(t, err)
		unsupported, err := asn1.MarshalWithParams(`{"v":2,"purpose":"ade"}`, "utf8")
		require.NoError(t, err)
		cert.Extensions[0].Value = unsupported
		_, err = ParseAppleMDMCertificateBindingExtension(cert)
		require.Error(t, err)
	})
}
