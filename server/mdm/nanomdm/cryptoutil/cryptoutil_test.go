package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

const mdmSignatureHeader1 = "MIAGCSqGSIb3DQEHAqCAMIACAQExCzAJBgUrDgMCGgUAMIAGCSqGSIb3DQEHAQAAoIIDIzCCAx8wggIHoAMCAQICAQQwDQYJKoZIhvcNAQELBQAwPTETMBEGCgmSJomT8ixkARkWA2NvbTEVMBMGCgmSJomT8ixkARkWBUd1c3RvMQ8wDQYDVQQDDAZNRE0gQ0EwHhcNMjEwOTE4MTg0NTA1WhcNMjIwOTE4MTg0NTA1WjAAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA8Weag+4AQFkLrgm2/lZCdjGj5KC2rbIKdBdfExxaFWmvTtNCdXWyd5eROboRuEG/D1Zun0WKaKc1/emikBhnXP4qzEnNobx1OOfzeR1ZiazwftgAKrDZK6e4IJo15x8juRZvbjfKAQV+fw6TIGe4COUKpBtJo1idxJzI6OO2pQ6tvfzxhvbeD8VtYoHFgTXmBDHqUjmixdM+RIDUqReemaTeK5ybWTw3ZrydR7lM+I92Y9x/sRSxTODjgcczmprMVFl7a/d7biuqJtxg/RRVA85LWE3Gl+3BaVi9TC8xzaVioC++RmbXe3Z5qHmm+fkhfIzHksBW0Yn0DmWZRoWpgwIDAQABo2cwZTAOBgNVHQ8BAf8EBAMCB4AwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHQYDVR0OBBYEFCxqnx50ZpbKaED6AAsxSScMguy6MB8GA1UdIwQYMBaAFBoyVn803d9H43znmXRJGmE066VrMA0GCSqGSIb3DQEBCwUAA4IBAQAU0jY/wjNth2fJsp49hbhEUFFPJIvM9lS5cWmSX2Xg7cK1pzDZJktA5MAZaLxbYCqpM9HegE3WhpyzaFRcIpBWV6T4R70gWbKcwn7WzAII0TBbDD4nZz2tO0kdLXA4LPyPjm/tJxzNvLfYmVNF61oImU2KXT/zp7rXOLU3KhkA4cWN9TApClTIZqlzr64T07HUA94S2ee9ia8/U2ITOswtYrGNYmky1PA9/GlcGaxm5LkthmIq4qh5/e8J8rfSXvz7GVuVqoZOBPVTQkBChG6ANCtTr8nniRIv+3L3042XjclVFj5mcLsXO5EN/v0i11ICcLs2SRJAF058CPLS7azgMYIBaTCCAWUCAQEwQjA9MRMwEQYKCZImiZPyLGQBGRYDY29tMRUwEwYKCZImiZPyLGQBGRYFR3VzdG8xDzANBgNVBAMMBk1ETSBDQQIBBDAJBgUrDgMCGgUAMA0GCSqGSIb3DQEBBQUABIIBAABiveq4A69qvK2FjCMdhm6o9aBPfTw8WiJU9I6UppTbvw1+o2OBVLAOCXw46v1SIbj7Lhq5EDm3qXLD2xkF9zd5W43PvNZFleL735De+I1IeyXOvkmElOioipDNwrRpsET6vL2zwYlE0JZuGVhr2EU8ra3czy4eAbJwvV2xHLjpvqQJZh0LNvBc10sp7Q/99qpVdCXagUPJTh68Pcua51JiUWn0tDn0eaj083Yyx+I1XNR9opYuBEVz/LwFSsUGiB9zV7KbsLikajD2+Jmues5vS2jOrmCpV+yMN3uMa4lmOlgrQoi4l62edTo45zgnEZOUle0zT2pInMgML8KiWt8AAAAAAAA="
const mdmSignatureHeader2 = "MIAGCSqGSIb3DQEHAqCAMIACAQExCzAJBgUrDgMCGgUAMIAGCSqGSIb3DQEHAQAAoIIDITCCAx0wggIFoAMCAQICASIwDQYJKoZIhvcNAQELBQAwOzELMAkGA1UEBhMCVVMxDTALBgNVBAoTBHNjZXAxEDAOBgNVBAsTB1NDRVAgQ0ExCzAJBgNVBAMTAmNhMB4XDTIxMDUyODA1MDkyM1oXDTMxMDUyNjA1MDkyM1owADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAO1oBJs+BT6daIs7bUWib7Ji1XGpDJ9e5uPtBJ3SVj2eIStDcl9bJj5JPle8HF+WI1HQ7yIT15zu3GSN6xO6gATAk1j/vv0JFimKXa9Q017mvg5ACUcRScY1mnya7pYL9b5C/b02ubOglEMCLnekhDKJhlVTwbdC0n0ZFr6aDWZlIxHFIym28Et9F+JeRJSRCAR6jybr+nRPavxPhGnbrImBM0z6HkzbjlHDlc5g3TOrj67bjz2nT+i6isEyj1stsOeDXH4Ij3z6A3jVW12/XAsHVoaC1KqmzXV2SWuOT9oXnlxs5NxoYJYQC+XN1x604cKauLNZFHjCFOFeeubVHucCAwEAAaNnMGUwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMCMB0GA1UdDgQWBBQpqTV2kEzUfYz+4Mws1S6/D2hGnzAfBgNVHSMEGDAWgBT5TUXoM3msHybCjct8RyHuOpPr6DANBgkqhkiG9w0BAQsFAAOCAQEAMdbAM0+753sYs9erWV+QW8PTYHHTlrGaOU9VB3QaY0TG6iy/k6+l/6rYWQGW+RyElxju6xof9BVYgKb17EB0vm635PSFrAviyp495vLSFesvUUkUr7ALKC7f6H8ud3JIRhycyyih30YSlZrCwduMNjocYXPZN36hsyLeG3c977RGdHLrXFAMCvOSqHLktj/pYChR95XwrKVBaX2jtsk9B9UcDmubccPjROUKWHn3k5f2R+EJv9WVmFqNt7doG4nAAZx0ktDdgzVO7eWz/RwfFZlA9Qtm4+gG5eQd8e8HVefcCNOT/HcjfBblsOM7jzQlIvxvNss8haQ6TC//Fk/0rTGCAWcwggFjAgEBMEAwOzELMAkGA1UEBhMCVVMxDTALBgNVBAoTBHNjZXAxEDAOBgNVBAsTB1NDRVAgQ0ExCzAJBgNVBAMTAmNhAgEiMAkGBSsOAwIaBQAwDQYJKoZIhvcNAQEFBQAEggEA5h26uR27aLsghfAuDGe0mefTcxnPbqpiOFzh8t7WbK69giGQIIfs/NVqdFvYn210l2y5cAyHFVU81w6MmvqjHQJqVLptyYVDXgYy0/tsRc6Tbc+5kKD3GEkP8Z4uMCn75/8QlXFCj5DTVNWTZ8Vn/Itns51z+JYccOcsj9TnLX+0aaLz3I8X+r31Ftlx3UP6keW7HDo8m6JeGN4P3t51Dqjt08zyu72yR/gxaS91SHROypPAwMbiKqaxGiAPChgACmZDf0c/+g5T6Q0qc++cEAP9n+ViSjXyoDtS3YbfWxiiJaiieXLAiPjU9eVcXajEAwJUOJtTfECrNd5pgNpLBgAAAAAAAA=="
const mdmSignatureBody2 = "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPCFET0NUWVBFIHBsaXN0IFBVQkxJQyAiLS8vQXBwbGUvL0RURCBQTElTVCAxLjAvL0VOIiAiaHR0cDovL3d3dy5hcHBsZS5jb20vRFREcy9Qcm9wZXJ0eUxpc3QtMS4wLmR0ZCI+CjxwbGlzdCB2ZXJzaW9uPSIxLjAiPgo8ZGljdD4KCTxrZXk+U3RhdHVzPC9rZXk+Cgk8c3RyaW5nPklkbGU8L3N0cmluZz4KCTxrZXk+VURJRDwva2V5PgoJPHN0cmluZz5ENEQ1NjA4My1CRkIwLTUwNDctOEJERi0yNzI3QkFDREM1OUI8L3N0cmluZz4KPC9kaWN0Pgo8L3BsaXN0Pgo="

func TestPKCS7ParseTagLengthError(t *testing.T) {
	// Regression test: Older versions of the library might return this BER tag length error.
	p7signature, _ := base64.StdEncoding.DecodeString(mdmSignatureHeader1)

	_, err := pkcs7.Parse(p7signature)

	if err != nil {
		t.Error("pkcs7.Parse() failed with error:", err)
	}
}

func TestVerifyMdmSignature(t *testing.T) {
	body, err := base64.StdEncoding.DecodeString(mdmSignatureBody2)
	if err != nil {
		t.Error(err)
	}
	_, err = VerifyMdmSignature(mdmSignatureHeader2, body)
	if err != nil {
		t.Error(err)
	}
}

func TestVerifyMdmSignatureIgnoringExpiry(t *testing.T) {
	// SCEP enrollments use RSA 2048 device keys, ACME enrollments use EC P-384.
	signerKeys := map[string]func(t *testing.T) crypto.Signer{
		"rsa": func(t *testing.T) crypto.Signer {
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)
			return key
		},
		"p384": func(t *testing.T) crypto.Signer {
			key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
			require.NoError(t, err)
			return key
		},
	}
	for name, newKey := range signerKeys {
		t.Run(name, func(t *testing.T) {
			body := []byte("<plist>checkin</plist>")
			now := time.Now()

			expiredCert, expiredHeader := signMdmBody(t, newKey(t), body, now.Add(-24*time.Hour), now.Add(-time.Hour))
			_, validHeader := signMdmBody(t, newKey(t), body, now.Add(-time.Hour), now.Add(24*time.Hour))
			_, notYetValidHeader := signMdmBody(t, newKey(t), body, now.Add(time.Hour), now.Add(24*time.Hour))

			_, err := VerifyMdmSignature(expiredHeader, body)
			var signingTimeErr *pkcs7.SigningTimeNotValidError
			require.ErrorAs(t, err, &signingTimeErr)

			cert, err := VerifyMdmSignatureIgnoringExpiry(expiredHeader, body)
			require.NoError(t, err)
			require.Equal(t, expiredCert.Raw, cert.Raw)
			require.Equal(t, expiredCert.NotAfter, cert.NotAfter, "returned cert must keep its real validity period")

			_, err = VerifyMdmSignatureIgnoringExpiry(notYetValidHeader, body)
			require.NoError(t, err)

			_, err = VerifyMdmSignatureIgnoringExpiry(validHeader, body)
			require.NoError(t, err)

			_, err = VerifyMdmSignatureIgnoringExpiry(expiredHeader, []byte("<plist>tampered</plist>"))
			var digestErr *pkcs7.MessageDigestMismatchError
			require.ErrorAs(t, err, &digestErr)
		})
	}
}

func signMdmBody(t *testing.T, key crypto.Signer, body []byte, notBefore, notAfter time.Time) (*x509.Certificate, string) {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "device"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	sd, err := pkcs7.NewSignedData(body)
	require.NoError(t, err)
	require.NoError(t, sd.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
	sd.Detach()
	sig, err := sd.Finish()
	require.NoError(t, err)

	return cert, base64.StdEncoding.EncodeToString(sig)
}
