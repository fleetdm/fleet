package mdmcrypto

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name        string
		cert        []byte
		key         []byte
		profile     []byte
		expectError bool
	}{
		{
			name:        "SCEP not set",
			profile:     []byte("profile data"),
			expectError: true,
		},
		{
			name:        "Error with invalid certificate",
			cert:        []byte("foo"),
			key:         []byte("bar"),
			profile:     []byte("profile data"),
			expectError: true,
		},
		{
			name:        "Successful signing",
			cert:        testCert,
			key:         testKey,
			profile:     []byte("profile data"),
			expectError: false,
		},
	}

	ds := new(mdmmock.MDMAppleStore)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
				_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				require.ElementsMatch(t, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, assetNames)
				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
					fleet.MDMAssetCACert: {Value: tc.cert},
					fleet.MDMAssetCAKey:  {Value: tc.key},
				}, nil
			}
			result, err := Sign(context.Background(), tc.profile, ds)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

var (
	testCert = []byte(`-----BEGIN CERTIFICATE-----
MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUx
EzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhv
cml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoM
ClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VH
cJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/U
fa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0
bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj
6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JS
kPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG
9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVj
E26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4
QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSw
VeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTf
UNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSn
dAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0H
MOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLX
KLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6Clie
TTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py
1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk
8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=
-----END CERTIFICATE-----`)

	testKey = []byte(testingKey(`-----BEGIN RSA TESTING KEY-----
MIIEogIBAAKCAQEApIdwfwSNK+nc2bggcAXEb0Nmbz7SHUHBTQ5fRUdwn4J/5/qH
Rdgxwyj0sUSmdQlFEyMqsziaQjvk0gi580E34au2qpM9OJ3zB84a79R9rkQRTWFv
aPsCIyWkzqLT5nKxdI2j5Vz9w1lPQ9d/I60mdibqn1ZkzzggPW3Z5rRsvALXZxMq
/mjJtUa0keHvpNHEBHcV5D35JjfS9QDFibqhbJiGjytF0q1RJ7XWlePo6HHF9Nme
xJafiWjS4GldqLrde2J5NWnPviG1I0LVu6cga/v/+bkXrSwfkysXklKQ/F8J+Cqy
SLzggINJE2W+lP54w+LKAAAnzNjzBhRdFgVhNQIDAQABAoIBAAtUbFHC3XnVq+iu
PkWYkBNdX9NvTwbGvWnyAGuD5OSHFwnBfck4fwzCaD9Ay/mpPsF3nXwj/LNs7m/s
O+ndZty6d2S9qOyaK98wuTgkuNbkRxC+Ee73wgjrkbLNEax/32p4Sn4D7lGid8vj
LhUl2k0ult+MEnsWkVnJk8TITeiQaT2AHhMr3HKdaI86hJJfam3wEBiLBglnnKqA
TInMqHoudnFOn/C8iVCFuHCE0oo1dMalbc4rlZuRBqezVhbSMWPLypMVXQb7eixM
ScJ3m8+DooGDSIe+EW/afhN2VnFbrhQC9/DlxGfwTwsUseWv7pgp53ufyyAzzydn
2plW/4ECgYEA1Va5RzSUDxr75JX003YZiBcYrG268vosiNYWRhE7frvn5EorZBRW
t4R70Y2gcXA10aPHzpbq40t6voWtpkfynU3fyRzbBmwfiWLEgckrYMwtcNz8nhG2
ETAg4LXO9CufbwuDa66h76TpkBzQVNc5TSbBUr/apLDWjKPMz6qW7VUCgYEAxW4K
Yqp3NgJkC5DhuD098jir9AH96hGhUryOi2CasCvmbjWCgWdolD7SRZJfxOXFOtHv
7Dkp9glA1Cg/nSmEHKslaTJfBIWK+5rqVD6k6kZE/+4QQWQtUxXXVgGINnGrnPvo
6MlRJxqGUtYJ0GRTFJP4Py0gwuzf5BMIwe+fpGECgYAOhLRfMCjTTlbOG5ZpvaPH
Kys2sNEEMBpPxaIGaq3N1iPV2WZSjT/JhW6XuDevAJ/pAGhcmtCpXz2fMaG7qzHL
mr0cBqaxLTKIOvx8iKA3Gi4NfDyE1Ve6m7fhEv5eh4l2GSZ8cYn7sRFkCVH0NCFm
KrkFVKEgjBhNwefySf2zcQKBgHDVPgw7nlv4q9LMX6RbI98eMnAG/2XZ45gUeWcA
tAeBX3WXEVoBjoxDBwuJ5z/xjXHbb8JSvT+G9E0MH6cjhgSYb44aoqFD7TV0yP2S
u8/Ej0SxewrURO8aKXJW99Edz9WtRuRbwgyWJTSMbRlzbOPy2UrJ8NJWbHK9yiCE
YXmhAoGAA3QUiCCl11c1C4VsF68Fa2i7qwnty3fvFidZpW3ds0tzZdIvkpRLp5+u
XAJ5+zStdEGdnu0iXALQlY7ektawXguT/zYKg3nfS9RMGW6CxZotn4bqfQwDuttf
b1xn1jGQd/o0xFf9ojpDNy6vNojidQGHh6E3h0GYvxbnQmVNq5U=
-----END RSA TESTING KEY-----`))
)

// prevent static analysis tools from raising issues due to detection of private key
// in code.
func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }
