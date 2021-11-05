package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestConfigRoundtrip(t *testing.T) {
	// This test verifies that a config can be roundtripped through yaml.
	// Doing so ensures that config_dump will provide the correct config.
	// Newly added config values will automatically be tested in this
	// function because of the reflection on the config struct.

	cmd := &cobra.Command{}
	// Leaving this flag unset means that no attempt will be made to load
	// the config file
	cmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")
	man := NewManager(cmd)

	// Use reflection magic to walk the config struct, setting unique
	// values to be verified on the roundtrip. Note that bools are always
	// set to true, which could false positive if the default value is
	// true.
	original := &FleetConfig{}
	v := reflect.ValueOf(original)
	for conf_index := 0; conf_index < v.Elem().NumField(); conf_index++ {
		conf_v := v.Elem().Field(conf_index)
		for key_index := 0; key_index < conf_v.NumField(); key_index++ {
			key_v := conf_v.Field(key_index)
			switch key_v.Interface().(type) {
			case string:
				switch conf_v.Type().Field(key_index).Name {
				case "TLSProfile":
					// we have to explicitly set value for this key as it will only
					// accept intermediate or modern
					key_v.SetString(TLSProfileModern)
				default:
					key_v.SetString(v.Elem().Type().Field(conf_index).Name + "_" + conf_v.Type().Field(key_index).Name)
				}
			case int:
				key_v.SetInt(int64(conf_index*100 + key_index))
			case bool:
				key_v.SetBool(true)
			case time.Duration:
				d := time.Duration(conf_index*100 + key_index)
				key_v.Set(reflect.ValueOf(d))
			}
		}
	}

	// Marshal the generated config
	buf, err := yaml.Marshal(original)
	require.Nil(t, err)
	t.Log(string(buf))

	// Manually load the serialized config
	man.viper.SetConfigType("yaml")
	err = man.viper.ReadConfig(bytes.NewReader(buf))
	require.Nil(t, err)

	// Ensure the read config is the same as the original
	assert.Equal(t, *original, man.LoadConfig())
}

func TestToTLSConfig(t *testing.T) {
	dir := t.TempDir()
	caFile, certFile, keyFile, garbageFile := filepath.Join(dir, "ca"),
		filepath.Join(dir, "cert"),
		filepath.Join(dir, "key"),
		filepath.Join(dir, "garbage")
	require.NoError(t, os.WriteFile(caFile, testCA, 0600))
	require.NoError(t, os.WriteFile(certFile, testCert, 0600))
	require.NoError(t, os.WriteFile(keyFile, testKey, 0600))
	require.NoError(t, os.WriteFile(garbageFile, []byte("zzzz"), 0600))

	cases := []struct {
		name        string
		in          TLS
		errContains string
	}{
		{"zero", TLS{}, ""},
		{"invalid file", TLS{TLSCA: "/no/such/file"}, "no such file"},
		{"CA", TLS{TLSCA: caFile}, ""},
		{"invalid CA content", TLS{TLSCA: garbageFile}, "failed to append PEM"},
		{"CA invalid cert", TLS{TLSCA: caFile, TLSCert: "/no/such/file"}, "no such file"},
		{"CA invalid key", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: "/no/such/file"}, "no such file"},
		{"CA cert key", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: keyFile}, ""},
		{"CA invalid cert content", TLS{TLSCA: caFile, TLSCert: garbageFile, TLSKey: keyFile}, "failed to find any PEM data"},
		{"CA invalid key content", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: garbageFile}, "failed to find any PEM data"},
		{"CA cert key server", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: keyFile, TLSServerName: "abc"}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.in.ToTLSConfig()
			if c.errContains != "" {
				require.Error(t, err)
				require.Nil(t, got)
				require.Contains(t, err.Error(), c.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			// root ca is required if TLSCA is set
			if c.in.TLSCA != "" {
				require.NotNil(t, got.RootCAs)
			} else {
				require.Nil(t, got.RootCAs)
			}
			require.Equal(t, got.ServerName, c.in.TLSServerName)
			if c.in.TLSCert != "" {
				require.Len(t, got.Certificates, 1)
			} else {
				require.Nil(t, got.Certificates)
			}
		})
	}
}

var (
	testCA = []byte(`-----BEGIN CERTIFICATE-----
MIIFSzCCAzOgAwIBAgIUf4lOcb9bkN2+u6FjWL0fSFCjGGgwDQYJKoZIhvcNAQEL
BQAwNTETMBEGA1UECgwKUmVkaXMgVGVzdDEeMBwGA1UEAwwVQ2VydGlmaWNhdGUg
QXV0aG9yaXR5MB4XDTIxMTAxOTEyNTEwNloXDTMxMTAxNzEyNTEwNlowNTETMBEG
A1UECgwKUmVkaXMgVGVzdDEeMBwGA1UEAwwVQ2VydGlmaWNhdGUgQXV0aG9yaXR5
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA02LNfNKjI/PwV4F2CVix
vVfFN41yxMKYkapTrvC1nc7lVmG5oxxgOIUpFT+7xj0+h2bBqR+t3eiFiaudz3Yc
9eG2J7BTtMST9QmQtNEyeC17TZxf4XB2EA68dYC24XaHBnSFsPg8/axlIVi1Hz7b
QmDRNY/X3cc3nzGxuuk3NnSN7s1UlKnZ1v0YZGwWhYD3iAv7kQcI3WYF0TF0nc2a
OXb68/AOghq9Z9zLk1ULIfTmT0fcJRsFssWClF7E378PSk0qjB6NEKADVyWq3d2g
8ValKmbKvAacsGxb2EXAPCJsBil0Sv7jAsl1hVfMCBwj6LfPKvn7/K8vbKz7Gtrw
COWVJtzaBrKzpjOTXQp9RnuqlDUZackTmn9hlCMLgapEC+j7PNvS8cyAbOz9bpEk
wdF/wrvUVsJc74+MXzEK7DWBKD2lP9nvY+0DrYJ/55KH1wbIH1RncLm6s6M4Zc9L
YfaeTuklimAOlx8WvuYQUJpxTh6gT4xWqZG2p8IcjxVp2Sl7eYtlaE/u7Ixc+Bfd
QpTaBXrtcQzttPNiSZM8b+nNL05p+LxtSVAYUu1Yc0hWBHJBb/dkDibOU3Mi8Aio
bvpsBp1RLfXSrRMOpXS3w4G1THrhC4IC1KkUbZ8EQaBlwa7mlwV8hxZOjJQ7Mf4D
Z8WEh1j/XH/zlKVJon2aUWUCAwEAAaNTMFEwHQYDVR0OBBYEFIDJJVTvQCl1vMIi
246T25FZVBsWMB8GA1UdIwQYMBaAFIDJJVTvQCl1vMIi246T25FZVBsWMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggIBAGZqxsaleZqmljrqrpL5JxoQ
G9/9tvfw5WYqeJ6r8s86HfxaqsEUemzSBb7HFJS42Ik6ghd32d62wp7xLxtQY8As
jvU9YZ2s42tSWgxch8kY/kgCjwsqTFViWmyxmc05TxulRr8BonIo8YAU6/5kBam+
sV5nfbBse5i9+nQqmjzVI7lVp7lIk+T9T4UsdH/mtbWv8cJjCBzbyObU+V9kjTSQ
O+cshOn59IMRvAkySKIHvm7keO4skazo2RMjdME9KW/ydc7iQ9YC0+MiDQF+eIAP
a/SGdTD8W/WNXT1rtD4DyTEZK1modAI7KukkrTwlaTW0GwssLq5TpwzQKK5W/ANZ
SU44yILArQrWZgXXxBBfGAH/asd4JgIxal/iM0hlYh6WYdSUa/QzJFFRngtE52jL
M1sTsUgXjItspH79oUD+my4ioDv6r2CAnlxl2MvqGzfBgItb5yq3bBwxNe/qOzWR
PbKbp3UvlzMbbpbeJHO2NHnu7Hha9mV3yr9+lsTv2SFeKGqFRbC7v+9kSDu6eOyC
lnARbzReZyZiYr9vCTxH76wCyUBBg7p59ZriBw0yaXvXcr4cO8IUPx4aPe9nHkbC
8G/rnKycuGGIDjslRTOJodxf2ud2UPYUTZDBi1QoV4+jzWKUjUxuHuN2WIwxnXKB
cJap0OI7VFpOjIJLzXRQ
-----END CERTIFICATE-----`)

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
