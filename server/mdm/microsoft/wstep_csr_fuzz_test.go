package microsoft_mdm

import (
	"bytes"
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzParseCertificateRequestFromWindowsDevice(f *testing.F) {
	for _, encoded := range csrBase64Array {
		der := fromBase64(encoded)
		f.Add(der)
		f.Add(bytes.ReplaceAll(der, []byte("Common Name"), []byte("Common!Name")))
		f.Add(bytes.ReplaceAll(der, []byte("Common Name"), []byte("Common\x00Name")))
	}
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x80})
	f.Add([]byte{0x30, 0x84, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, der []byte) {
		if len(der) > 64<<10 {
			t.Skip("CSR exceeds fuzz input budget")
		}
		got, err := ParseCertificateRequestFromWindowsDevice(der)
		if err != nil {
			return
		}
		require.Equal(t, der, got.Raw)

		// Windows permits subject characters rejected by the standard parser.
		want, err := x509.ParseCertificateRequest(der)
		if err != nil {
			return
		}
		require.Equal(t, want.RawSubject, got.RawSubject)
		// The fork and newer Go versions represent opaque subject attributes differently.
		wantSubject, gotSubject := want.Subject, got.Subject
		wantSubject.Names, gotSubject.Names = nil, nil
		require.Equal(t, wantSubject, gotSubject)
		require.Equal(t, want.PublicKey, got.PublicKey)
		require.Equal(t, want.SignatureAlgorithm, got.SignatureAlgorithm)
		require.Equal(t, want.Extensions, got.Extensions)
		require.Equal(t, want.DNSNames, got.DNSNames)
		require.Equal(t, want.EmailAddresses, got.EmailAddresses)
		require.Equal(t, want.IPAddresses, got.IPAddresses)
		require.Equal(t, want.URIs, got.URIs)
	})
}
