package service

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dsig "github.com/russellhaering/goxmldsig"
)

// kitlogAdapter adapts kitlog.Logger to saml logger.Interface
type kitlogAdapter struct {
	logger kitlog.Logger
}

func (k *kitlogAdapter) Printf(format string, v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprintf(format, v...))
}

func (k *kitlogAdapter) Print(v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprint(v...))
}

func (k *kitlogAdapter) Println(v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprintln(v...))
}

func (k *kitlogAdapter) Fatal(v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprint(v...))
	os.Exit(1)
}

func (k *kitlogAdapter) Fatalf(format string, v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (k *kitlogAdapter) Fatalln(v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprintln(v...))
	os.Exit(1)
}

func (k *kitlogAdapter) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}

func (k *kitlogAdapter) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}

func (k *kitlogAdapter) Panicln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}

// Hard-coded certificate and key for Okta conditional access POC
// TODO: Move to configuration/database in production
var oktaDeviceHealthKey = func() crypto.PrivateKey {
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

var oktaDeviceHealthCert = func() *x509.Certificate {
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

// Hard-coded Okta SP metadata for Okta conditional access POC
// TODO: Store in database and allow configuration per tenant
var oktaServiceProviderMetadata = func() *saml.EntityDescriptor {
	// Parse Okta's SP metadata from the IdP configuration
	metadataXML := `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor entityID="https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"><md:SPSSODescriptor AuthnRequestsSigned="true" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><md:KeyDescriptor use="encryption"><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>MIIDqDCCApCgAwIBAgIGAZXsT7aXMA0GCSqGSIb3DQEBCwUAMIGUMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEcMBoGCSqGSIb3DQEJ
ARYNaW5mb0Bva3RhLmNvbTAeFw0yNTAzMzExMzA1NDFaFw0zNTAzMzExMzA2NDFaMIGUMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsG
A1UECgwET2t0YTEUMBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEc
MBoGCSqGSIb3DQEJARYNaW5mb0Bva3RhLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIS/AMr00GVLTHWnufTZg9sWjJhEkEawLoSRtMPZhJRPi/8rKsKk0fiYK6YKHpiY+iL4kle0
NHVMAQhk6vC4wmiaKMy8iEZxJB2gWLO/Xk6b+Vaa1Fu4xg+wWb61ue46HGRhvhHG3eHtz8NOLao4
2DRCjbghCv+qRDcfgei/IrrTUmSJDUMXNMtaQbg+dOMeQRbgfkz2x6LI/TeBKghIGHIYRKzebcH6
kr1XtgVapG+X6NccjL4FmIvfITpOK6+B3wdszEH5HUicMdZEt/8yLO00kJZhxRVCvK0LbzYEHFx5
ftIyBCB6iwIZ9eECf4p87UxOfe0AD0NAdm/BR+dr1psCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEA
Wzh9U6/I5G/Uy/BoMTv3lBsbS6h7OGUE2kOTX5YF3+t4EKlGNHNHx1CcOa7kKb1Cpagnu3UfThly
nMVWcUemsnhjN+6DeTGpqX/GGpQ22YKIZbqFm90jS+CtLQQsi0ciU7w4d981T2I7oRs9yDk+A2ZF
9yf8wGi6ocy4EC00dCJ7DoSui6HdYiQWk60K4w7LPqtvx2bPPK9j+pmAbuLmHPAQ4qyccDZVDOaP
umSer90UyfV6FkY8/nfrqDk6tE8RyabI3o48Q4m12RoYcA3sZ3Ba3A4CzP7Q0uUFD6nMTqgq4ZeV
FqU+KJOed6qlzj7qy+u5l6CQeajLGdjUxFlFyw==</ds:X509Certificate></ds:X509Data></ds:KeyInfo><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes192-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc"/><md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes128-gcm"/><md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/><md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#tripledes-cbc"/></md:KeyDescriptor><md:KeyDescriptor use="signing"><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>MIIDqDCCApCgAwIBAgIGAZXsT7aXMA0GCSqGSIb3DQEBCwUAMIGUMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEcMBoGCSqGSIb3DQEJ
ARYNaW5mb0Bva3RhLmNvbTAeFw0yNTAzMzExMzA1NDFaFw0zNTAzMzExMzA2NDFaMIGUMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsG
A1UECgwET2t0YTEUMBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEc
MBoGCSqGSIb3DQEJARYNaW5mb0Bva3RhLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIS/AMr00GVLTHWnufTZg9sWjJhEkEawLoSRtMPZhJRPi/8rKsKk0fiYK6YKHpiY+iL4kle0
NHVMAQhk6vC4wmiaKMy8iEZxJB2gWLO/Xk6b+Vaa1Fu4xg+wWb61ue46HGRhvhHG3eHtz8NOLao4
2DRCjbghCv+qRDcfgei/IrrTUmSJDUMXNMtaQbg+dOMeQRbgfkz2x6LI/TeBKghIGHIYRKzebcH6
kr1XtgVapG+X6NccjL4FmIvfITpOK6+B3wdszEH5HUicMdZEt/8yLO00kJZhxRVCvK0LbzYEHFx5
ftIyBCB6iwIZ9eECf4p87UxOfe0AD0NAdm/BR+dr1psCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEA
Wzh9U6/I5G/Uy/BoMTv3lBsbS6h7OGUE2kOTX5YF3+t4EKlGNHNHx1CcOa7kKb1Cpagnu3UfThly
nMVWcUemsnhjN+6DeTGpqX/GGpQ22YKIZbqFm90jS+CtLQQsi0ciU7w4d981T2I7oRs9yDk+A2ZF
9yf8wGi6ocy4EC00dCJ7DoSui6HdYiQWk60K4w7LPqtvx2bPPK9j+pmAbuLmHPAQ4qyccDZVDOaP
umSer90UyfV6FkY8/nfrqDk6tE8RyabI3o48Q4m12RoYcA3sZ3Ba3A4CzP7Q0uUFD6nMTqgq4ZeV
FqU+KJOed6qlzj7qy+u5l6CQeajLGdjUxFlFyw==</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat><md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://dev-57988129.okta.com/sso/saml2/0oaqu0748bP2ELUlJ5d7" index="0" isDefault="true"/><md:AttributeConsumingService index="0"><md:ServiceName xml:lang="en">Fleet</md:ServiceName><md:RequestedAttribute FriendlyName="First Name" Name="firstName" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Last Name" Name="lastName" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Email" Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="true"/><md:RequestedAttribute FriendlyName="Mobile Phone" Name="mobilePhone" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri" isRequired="false"/></md:AttributeConsumingService></md:SPSSODescriptor><md:Organization><md:OrganizationName xml:lang="en">dev-57988129</md:OrganizationName><md:OrganizationDisplayName xml:lang="en">okta-dev-57988129</md:OrganizationDisplayName><md:OrganizationURL xml:lang="en">https://developer.okta.com</md:OrganizationURL></md:Organization></md:EntityDescriptor>`

	var metadata saml.EntityDescriptor
	if err := xml.Unmarshal([]byte(metadataXML), &metadata); err != nil {
		// This should never fail since metadataXML is hardcoded and valid
		// If it does fail, return empty descriptor which will cause errors later
		return &saml.EntityDescriptor{}
	}

	return &metadata
}()

// oktaDeviceHealthSessionProvider implements saml.SessionProvider
type oktaDeviceHealthSessionProvider struct {
	svc fleet.Service
}

func (o *oktaDeviceHealthSessionProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	// Extract email from request
	email := ""
	if req.Request.Subject != nil && req.Request.Subject.NameID != nil {
		email = req.Request.Subject.NameID.Value
	}

	// Phase 1: Always return valid session (always pass)
	return &saml.Session{
		NameID: email,
	}
}

// oktaDeviceHealthSPProvider implements saml.ServiceProviderProvider
type oktaDeviceHealthSPProvider struct {
	svc fleet.Service
}

func (o *oktaDeviceHealthSPProvider) GetServiceProvider(r *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	// Phase 1: Return hardcoded Okta SP metadata
	// TODO: Look up from database
	return oktaServiceProviderMetadata, nil
}

// getOktaDeviceHealthIDP creates and configures the SAML IdP
func getOktaDeviceHealthIDP(svc fleet.Service, baseURL string, logger kitlog.Logger) (*saml.IdentityProvider, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	metadataURL := *u
	metadataURL.Path = "/api/v1/fleet/okta/device_health/metadata"

	ssoURL := *u
	ssoURL.Path = "/api/v1/fleet/okta/device_health/sso"

	sessionProvider := &oktaDeviceHealthSessionProvider{svc: svc}
	spProvider := &oktaDeviceHealthSPProvider{svc: svc}

	// Use kitlog adapter to satisfy saml logger.Interface
	samlLogger := &kitlogAdapter{logger: kitlog.With(logger, "component", "saml-idp")}

	idp := &saml.IdentityProvider{
		Key:                     oktaDeviceHealthKey,
		SignatureMethod:         dsig.RSASHA256SignatureMethod,
		Logger:                  samlLogger,
		Certificate:             oktaDeviceHealthCert,
		MetadataURL:             metadataURL,
		SSOURL:                  ssoURL,
		ServiceProviderProvider: spProvider,
		SessionProvider:         sessionProvider,
	}

	return idp, nil
}

// //////////////////////////////////////////////////////////////////////////////
// GET /api/v1/fleet/okta/device_health/metadata
// //////////////////////////////////////////////////////////////////////////////

func makeOktaDeviceHealthMetadataHandler(svc fleet.Service, baseURL string, logger kitlog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idp, err := getOktaDeviceHealthIDP(svc, baseURL, logger)
		if err != nil {
			logger.Log("err", "error creating IdP", "details", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ServeMetadata writes XML directly to the response
		idp.ServeMetadata(w, r)
	}
}

// //////////////////////////////////////////////////////////////////////////////
// POST /api/v1/fleet/okta/device_health/sso
// //////////////////////////////////////////////////////////////////////////////

func makeOktaDeviceHealthSSOHandler(svc fleet.Service, baseURL string, logger kitlog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idp, err := getOktaDeviceHealthIDP(svc, baseURL, logger)
		if err != nil {
			logger.Log("err", "error creating IdP", "details", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ServeSSO handles the SAML AuthnRequest and sends back a response
		idp.ServeSSO(w, r)
	}
}
