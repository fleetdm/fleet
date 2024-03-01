package main

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
)

var key = func() crypto.PrivateKey {
	b, _ := pem.Decode([]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0OhbMuizgtbFOfwbK7aURuXhZx6VRuAs3nNibiuifwCGz6u9
yy7bOR0P+zqN0YkjxaokqFgra7rXKCdeABmoLqCC0U+cGmLNwPOOA0PaD5q5xKhQ
4Me3rt/R9C4Ca6k3/OnkxnKwnogcsmdgs2l8liT3qVHP04Oc7Uymq2v09bGb6nPu
fOrkXS9F6mSClxHG/q59AGOWsXK1xzIRV1eu8W2SNdyeFVU1JHiQe444xLoPul5t
InWasKayFsPlJfWNc8EoU8COjNhfo/GovFTHVjh9oUR/gwEFVwifIHihRE0Hazn2
EQSLaOr2LM0TsRsQroFjmwSGgI+X2bfbMTqWOQIDAQABAoIBAFWZwDTeESBdrLcT
zHZe++cJLxE4AObn2LrWANEv5AeySYsyzjRBYObIN9IzrgTb8uJ900N/zVr5VkxH
xUa5PKbOcowd2NMfBTw5EEnaNbILLm+coHdanrNzVu59I9TFpAFoPavrNt/e2hNo
NMGPSdOkFi81LLl4xoadz/WR6O/7N2famM+0u7C2uBe+TrVwHyuqboYoidJDhO8M
w4WlY9QgAUhkPyzZqrl+VfF1aDTGVf4LJgaVevfFCas8Ws6DQX5q4QdIoV6/0vXi
B1M+aTnWjHuiIzjBMWhcYW2+I5zfwNWRXaxdlrYXRukGSdnyO+DH/FhHePJgmlkj
NInADDkCgYEA6MEQFOFSCc/ELXYWgStsrtIlJUcsLdLBsy1ocyQa2lkVUw58TouW
RciE6TjW9rp31pfQUnO2l6zOUC6LT9Jvlb9PSsyW+rvjtKB5PjJI6W0hjX41wEO6
fshFELMJd9W+Ezao2AsP2hZJ8McCF8no9e00+G4xTAyxHsNI2AFTCQcCgYEA5cWZ
JwNb4t7YeEajPt9xuYNUOQpjvQn1aGOV7KcwTx5ELP/Hzi723BxHs7GSdrLkkDmi
Gpb+mfL4wxCt0fK0i8GFQsRn5eusyq9hLqP/bmjpHoXe/1uajFbE1fZQR+2LX05N
3ATlKaH2hdfCJedFa4wf43+cl6Yhp6ZA0Yet1r8CgYEAwiu1j8W9G+RRA5/8/DtO
yrUTOfsbFws4fpLGDTA0mq0whf6Soy/96C90+d9qLaC3srUpnG9eB0CpSOjbXXbv
kdxseLkexwOR3bD2FHX8r4dUM2bzznZyEaxfOaQypN8SV5ME3l60Fbr8ajqLO288
wlTmGM5Mn+YCqOg/T7wjGmcCgYBpzNfdl/VafOROVbBbhgXWtzsz3K3aYNiIjbp+
MunStIwN8GUvcn6nEbqOaoiXcX4/TtpuxfJMLw4OvAJdtxUdeSmEee2heCijV6g3
ErrOOy6EqH3rNWHvlxChuP50cFQJuYOueO6QggyCyruSOnDDuc0BM0SGq6+5g5s7
H++S/wKBgQDIkqBtFr9UEf8d6JpkxS0RXDlhSMjkXmkQeKGFzdoJcYVFIwq8jTNB
nJrVIGs3GcBkqGic+i7rTO1YPkquv4dUuiIn+vKZVoO6b54f+oPBXd4S0BnuEqFE
rdKNuCZhiaE2XD9L/O9KP1fh5bfEcKwazQ23EvpJHBMm8BGC+/YZNw==
-----END RSA PRIVATE KEY-----`))
	k, _ := x509.ParsePKCS1PrivateKey(b.Bytes)
	return k
}()

var cert = func() *x509.Certificate {
	b, _ := pem.Decode([]byte(`-----BEGIN CERTIFICATE-----
MIIDBzCCAe+gAwIBAgIJAPr/Mrlc8EGhMA0GCSqGSIb3DQEBBQUAMBoxGDAWBgNV
BAMMD3d3dy5leGFtcGxlLmNvbTAeFw0xNTEyMjgxOTE5NDVaFw0yNTEyMjUxOTE5
NDVaMBoxGDAWBgNVBAMMD3d3dy5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANDoWzLos4LWxTn8Gyu2lEbl4WcelUbgLN5zYm4ron8A
hs+rvcsu2zkdD/s6jdGJI8WqJKhYK2u61ygnXgAZqC6ggtFPnBpizcDzjgND2g+a
ucSoUODHt67f0fQuAmupN/zp5MZysJ6IHLJnYLNpfJYk96lRz9ODnO1Mpqtr9PWx
m+pz7nzq5F0vRepkgpcRxv6ufQBjlrFytccyEVdXrvFtkjXcnhVVNSR4kHuOOMS6
D7pebSJ1mrCmshbD5SX1jXPBKFPAjozYX6PxqLxUx1Y4faFEf4MBBVcInyB4oURN
B2s59hEEi2jq9izNE7EbEK6BY5sEhoCPl9m32zE6ljkCAwEAAaNQME4wHQYDVR0O
BBYEFB9ZklC1Ork2zl56zg08ei7ss/+iMB8GA1UdIwQYMBaAFB9ZklC1Ork2zl56
zg08ei7ss/+iMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBAAVoTSQ5
pAirw8OR9FZ1bRSuTDhY9uxzl/OL7lUmsv2cMNeCB3BRZqm3mFt+cwN8GsH6f3uv
NONIhgFpTGN5LEcXQz89zJEzB+qaHqmbFpHQl/sx2B8ezNgT/882H2IH00dXESEf
y/+1gHg2pxjGnhRBN6el/gSaDiySIMKbilDrffuvxiCfbpPN0NRRiPJhd2ay9KuL
/RxQRl1gl9cHaWiouWWba1bSBb2ZPhv2rPMUsFo98ntkGCObDX6Y1SpkqmoTbrsb
GFsTG2DLxnvr4GdN1BSr0Uu/KV3adj47WkXVPeMYQti/bQmxQB8tRFhrw80qakTL
UzreO96WzlBBMtY=
-----END CERTIFICATE-----`))
	c, _ := x509.ParseCertificate(b.Bytes)
	return c
}()

var spMeta = func() *saml.EntityDescriptor {
	var res saml.EntityDescriptor
	err := xml.Unmarshal([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor entityID="https://www.okta.com/saml2/service-provider/spmgaikhxteyxumoajkg" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"><md:SPSSODescriptor AuthnRequestsSigned="true" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><md:KeyDescriptor use="encryption"><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>MIIDqjCCApKgAwIBAgIGAY2kcw61MA0GCSqGSIb3DQEBCwUAMIGVMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFjAUBgNVBAMMDXRyaWFsLTE5MzA2ODExHDAaBgkqhkiG9w0B
CQEWDWluZm9Ab2t0YS5jb20wHhcNMjQwMjEzMjE0OTIwWhcNMzQwMjEzMjE1MDIwWjCBlTELMAkG
A1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDTAL
BgNVBAoMBE9rdGExFDASBgNVBAsMC1NTT1Byb3ZpZGVyMRYwFAYDVQQDDA10cmlhbC0xOTMwNjgx
MRwwGgYJKoZIhvcNAQkBFg1pbmZvQG9rdGEuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAoNxTYBkURtCx75KqjDIzh0q61l7QXYjtFfy5OPDp9h6I3aDyUKdJtR3DAn2R3TjAQu8/
336+/76SsC3fBBEkXd4DWJ9dzd8zxX0MDYg2VL6/L/zrCq9+oK8UDGGGh6EhpKbc8ZOysyoX0sbZ
BraZxtCuRkmZYMfJVudVfCTiJn/2hy3NdfP33AN+d+Ercr5oJpftJwzAqvZZ9YzvytoPV26k5yQv
p3sPg/mgp6mMYOVw9LNnPvRHAOjks3Dlh/h0ZQ9vhUvPvu2pVV4nX7+1wfb/6cBt7RSmnimOoV2+
1b0Bttg87GRjgfbjT7ejijbxqVwSVslFedO2PETLHPOBywIDAQABMA0GCSqGSIb3DQEBCwUAA4IB
AQBiey581q7M2wBGWchyyoSYQTuN0YnHhx3KEEcq2evZH/TmWJ1Yjx8MSCPtTgUzDkpv6aAnpcF1
/1y1xW7Rm4T9is6BaXWSKzm3e2m4217/IdgJ3NfLZE52imRdaZHkVdiM1//NlC5EG7bRBr002ciR
6ShKHatYFpA8HDW2RIbiDg4NHI4HN5sXGWf+evPrpdbREHN0MHo4BMe3MM0SDE+/sUC3nhTmc6D1
JYmdRUj+GvUv0cqieUESf/A9yMDsdjTcgNzStGU93JXhJDJ/GUZpveHmPY7GH+0k8CW36LZrtxKG
J6k/4lrVU/5lJWohLx7BSfGCc9OrH6DYG++YSlHy</ds:X509Certificate></ds:X509Data></ds:KeyInfo><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes192-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes128-gcm"/><md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#tripledes-cbc"/></md:KeyDescriptor><md:KeyDescriptor use="signing"><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>MIIDqjCCApKgAwIBAgIGAY2kcw61MA0GCSqGSIb3DQEBCwUAMIGVMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFjAUBgNVBAMMDXRyaWFsLTE5MzA2ODExHDAaBgkqhkiG9w0B
CQEWDWluZm9Ab2t0YS5jb20wHhcNMjQwMjEzMjE0OTIwWhcNMzQwMjEzMjE1MDIwWjCBlTELMAkG
A1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDTAL
BgNVBAoMBE9rdGExFDASBgNVBAsMC1NTT1Byb3ZpZGVyMRYwFAYDVQQDDA10cmlhbC0xOTMwNjgx
MRwwGgYJKoZIhvcNAQkBFg1pbmZvQG9rdGEuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAoNxTYBkURtCx75KqjDIzh0q61l7QXYjtFfy5OPDp9h6I3aDyUKdJtR3DAn2R3TjAQu8/
336+/76SsC3fBBEkXd4DWJ9dzd8zxX0MDYg2VL6/L/zrCq9+oK8UDGGGh6EhpKbc8ZOysyoX0sbZ
BraZxtCuRkmZYMfJVudVfCTiJn/2hy3NdfP33AN+d+Ercr5oJpftJwzAqvZZ9YzvytoPV26k5yQv
p3sPg/mgp6mMYOVw9LNnPvRHAOjks3Dlh/h0ZQ9vhUvPvu2pVV4nX7+1wfb/6cBt7RSmnimOoV2+
1b0Bttg87GRjgfbjT7ejijbxqVwSVslFedO2PETLHPOBywIDAQABMA0GCSqGSIb3DQEBCwUAA4IB
AQBiey581q7M2wBGWchyyoSYQTuN0YnHhx3KEEcq2evZH/TmWJ1Yjx8MSCPtTgUzDkpv6aAnpcF1
/1y1xW7Rm4T9is6BaXWSKzm3e2m4217/IdgJ3NfLZE52imRdaZHkVdiM1//NlC5EG7bRBr002ciR
6ShKHatYFpA8HDW2RIbiDg4NHI4HN5sXGWf+evPrpdbREHN0MHo4BMe3MM0SDE+/sUC3nhTmc6D1
JYmdRUj+GvUv0cqieUESf/A9yMDsdjTcgNzStGU93JXhJDJ/GUZpveHmPY7GH+0k8CW36LZrtxKG
J6k/4lrVU/5lJWohLx7BSfGCc9OrH6DYG++YSlHy</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat><md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://trial-1930681.okta.com/sso/saml2/0oabsqp1lb2Vs5eZ4697" index="0" isDefault="true"/><md:AttributeConsumingService index="0"><md:ServiceName xml:lang="en">Fleet test</md:ServiceName><md:RequestedAttribute FriendlyName="First Name" Name="firstName" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Last Name" Name="lastName" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Email" Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Mobile Phone" Name="mobilePhone" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="false"/></md:AttributeConsumingService></md:SPSSODescriptor><md:Organization><md:OrganizationName xml:lang="en">trial-1930681</md:OrganizationName><md:OrganizationDisplayName xml:lang="en">fleetdm-trial-1930681</md:OrganizationDisplayName><md:OrganizationURL xml:lang="en">https://fleetdm.com</md:OrganizationURL></md:Organization></md:EntityDescriptor>`),
		&res)
	if err != nil {
		panic(err)
	}
	return &res
}()

type identityProvider struct{}

func (i *identityProvider) GetServiceProvider(r *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	log.Printf("request id: %s", serviceProviderID)
	return spMeta, nil
}

func (i *identityProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	email := req.Request.Subject.NameID.Value
	log.Printf("auth email: %+v", email)

	if err := checkForEmail(email); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html>
		<body>
		<img src="/img/failure.png" style="width:80%; display: block;  margin-left: auto;  margin-right: auto;" />
		</body>
		</html>`)
		return nil
	}

	return &saml.Session{
		NameID: email,
	}
}

func getIDP() *saml.IdentityProvider {
	u, err := url.Parse("http://" + addr)
	if err != nil {
		panic(err)
	}
	metadataURL := *u
	metadataURL.Path = metadataURL.Path + "/metadata"
	ssoURL := *u
	ssoURL.Path = ssoURL.Path + "/sso"

	i := &identityProvider{}

	idp := &saml.IdentityProvider{
		Key:                     key,
		SignatureMethod:         dsig.RSASHA256SignatureMethod,
		Logger:                  log.New(os.Stderr, "idp: ", 0),
		Certificate:             cert,
		MetadataURL:             metadataURL,
		SSOURL:                  ssoURL,
		ServiceProviderProvider: i,
		SessionProvider:         i,
	}
	return idp
}
