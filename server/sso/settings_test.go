package sso

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var metadata = `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="http://www.okta.com/exka4zkf6dxm8pF220h7">
  <md:IDPSSODescriptor WantAuthnRequestsSigned="false" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDpDCCAoygAwIBAgIGAVtYB4c1MA0GCSqGSIb3DQEBCwUAMIGSMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxEzARBgNVBAMMCmRldi0xMzIwMzgxHDAaBgkqhkiG9w0BCQEW
DWluZm9Ab2t0YS5jb20wHhcNMTcwNDEwMTMyMTIwWhcNMjcwNDEwMTMyMjIwWjCBkjELMAkGA1UE
BhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDTALBgNV
BAoMBE9rdGExFDASBgNVBAsMC1NTT1Byb3ZpZGVyMRMwEQYDVQQDDApkZXYtMTMyMDM4MRwwGgYJ
KoZIhvcNAQkBFg1pbmZvQG9rdGEuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
p+FVPm80AKh7HyTrVA05NHz8tMKIjtt0TmbRmjp6Mol+jGLYb5ILzPKQAmdh//a0hEXTTsPNw5H1
35fl5auXI1n6i8ZnoGc9/Ym7gSWKz93plbu1i3QbxhGHZmKvO66Ba7ag6z6ko6kg9A8k2UK4+5O4
T0toRUZ54YkH/ugDtfhspjlF5NjNwktL4Dj/EOel5A9I11WnHb2l3tYZkl0/viKSBOHfraPlbFUS
aG8kduQkW7+4bY18JUqDcNtQEVvlz0zm+g3WsfNM/Bhi1bxkI0aDCyZpBsXedaMv4KbZzOudx6LS
epixcLHWku5idBVRXqQTGLQdNW/P1qGQgdAeTQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBvpAp8
ExV9Wr/Q3x7fvmcNj9qnkONjTFs5k4Qhh5Ms/adq9kM7IgEeY7s6ZksC1v5nuQOAFWOWgzZS3aX3
Tgl1fvPaZFmq+wcykUPnaBFbY2awRyOeIgdNbUjgr6fvi/D8xvgunFG4TIGqfS33O/+h9bXaCMQB
EyHJq5F+u/h/L8f6CYiEY21qpl9bjL3g+li1tQTP7FnxAR/uj5cUsBp1ZdVUSyEvCWg0hJx6NQQF
3lLbb1x1Xj1Y+GKFjnNudyai660kM02xI4D5kgjz9Yp0c7UQ0Qufnq8OzpdIrVHsw4NIzJpLtO8D
rICQDchR6/cxoQCkoyf+/YTpY492MafV</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://dev-132038.oktapreview.com/app/dev132038_1/exka4zkf6dxm8pF220h7/sso/saml"/><md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://dev-132038.oktapreview.com/app/dev132038_1/exka4zkf6dxm8pF220h7/sso/saml"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>
`

func TestParseMetadata(t *testing.T) {
	settings, err := ParseMetadata([]byte(metadata))
	require.Nil(t, err)

	assert.Equal(t, "http://www.okta.com/exka4zkf6dxm8pF220h7", settings.EntityID)
	require.Len(t, settings.IDPSSODescriptors, 1)
	assert.Len(t, settings.IDPSSODescriptors[0].NameIDFormats, 2)
	require.Len(t, settings.IDPSSODescriptors[0].KeyDescriptors, 1)
	assert.True(t, settings.IDPSSODescriptors[0].KeyDescriptors[0].KeyInfo.X509Data.X509Certificates[0].Data != "")
	require.Len(t, settings.IDPSSODescriptors[0].SingleSignOnServices, 2)
	assert.Equal(t,
		"https://dev-132038.oktapreview.com/app/dev132038_1/exka4zkf6dxm8pF220h7/sso/saml",
		settings.IDPSSODescriptors[0].SingleSignOnServices[0].Location,
	)
	assert.Equal(t,
		"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		settings.IDPSSODescriptors[0].SingleSignOnServices[0].Binding,
	)
}

func TestGetMetadata(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(metadata))
		require.NoError(t, err)
	}))
	xmlMetadata, err := getMetadata(ts.URL)
	require.NoError(t, err)
	settings, err := ParseMetadata(xmlMetadata)
	require.NoError(t, err)
	assert.Equal(t, "http://www.okta.com/exka4zkf6dxm8pF220h7", settings.EntityID)
	assert.Len(t, settings.IDPSSODescriptors, 1)
	assert.Len(t, settings.IDPSSODescriptors[0].NameIDFormats, 2)
	require.Len(t, settings.IDPSSODescriptors[0].KeyDescriptors, 1)
	assert.True(t, settings.IDPSSODescriptors[0].KeyDescriptors[0].KeyInfo.X509Data.X509Certificates[0].Data != "")
	require.Len(t, settings.IDPSSODescriptors[0].SingleSignOnServices, 2)
	assert.Equal(t,
		"https://dev-132038.oktapreview.com/app/dev132038_1/exka4zkf6dxm8pF220h7/sso/saml",
		settings.IDPSSODescriptors[0].SingleSignOnServices[0].Location,
	)
	assert.Equal(t,
		"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		settings.IDPSSODescriptors[0].SingleSignOnServices[0].Binding,
	)
}
