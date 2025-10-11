package okta

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
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dsig "github.com/russellhaering/goxmldsig"
)

// Service handles Okta conditional access integration
type Service struct {
	ds             fleet.Datastore
	nanoMDMStorage storage.AllStorage
	logger         kitlog.Logger
}

// NewService creates a new Okta service
func NewService(ds fleet.Datastore, nanoMDMStorage storage.AllStorage, logger kitlog.Logger) *Service {
	return &Service{
		ds:             ds,
		nanoMDMStorage: nanoMDMStorage,
		logger:         logger,
	}
}

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
	svc *Service
}

func (o *oktaDeviceHealthSessionProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	ctx := r.Context()

	// Extract email from request
	email := ""
	if req.Request.Subject != nil && req.Request.Subject.NameID != nil {
		email = req.Request.Subject.NameID.Value
	}

	if email == "" {
		o.renderRemediationPage(w, "No Email Provided", "Your account does not have an email address associated with it.", nil)
		return nil
	}

	// Try to get host from client certificate hash (more secure, cryptographically verified)
	var host *fleet.Host
	if certHash, ok := ctx.Value(clientCertHashKey).(string); ok && certHash != "" {
		level.Info(o.svc.logger).Log("msg", "looking up host by certificate hash", "hash", certHash)

		// Look up enrollment ID from certificate hash (same as MDM does)
		enrollmentID, err := o.svc.nanoMDMStorage.EnrollmentFromHash(ctx, certHash)
		if err != nil {
			level.Error(o.svc.logger).Log("msg", "failed to lookup enrollment from certificate hash", "hash", certHash, "err", err)
			o.renderRemediationPage(w, "Certificate Lookup Failed", "Unable to verify device certificate.", nil)
			return nil
		}

		if enrollmentID == "" {
			level.Error(o.svc.logger).Log("msg", "certificate not associated with any enrollment", "hash", certHash)
			o.renderRemediationPage(w, "Device Not Enrolled", "This certificate is not associated with an enrolled device.", nil)
			return nil
		}

		// The enrollment ID is the device UUID, use HostByIdentifier to get the host
		host, err = o.svc.ds.HostByIdentifier(ctx, enrollmentID)
		if err != nil {
			level.Error(o.svc.logger).Log("msg", "host not found for enrollment", "enrollment_id", enrollmentID, "err", err)
			o.renderRemediationPage(w, "Device Not Found", "Device is enrolled in MDM but not found in Fleet.", nil)
			return nil
		}

		level.Info(o.svc.logger).Log("msg", "found host by certificate", "host_id", host.ID, "hostname", host.Hostname, "enrollment_id", enrollmentID)
	} else {
		// Fallback: look up host ID by email (less secure, for testing without mTLS)
		level.Info(o.svc.logger).Log("msg", "no certificate serial, falling back to email lookup", "email", email)

		hostID, err := o.svc.ds.HostIDByEmail(ctx, email)
		if err != nil {
			o.renderRemediationPage(w, "Device Not Found", fmt.Sprintf("No device found for email: %s", email), nil)
			return nil
		}

		host, err = o.svc.ds.Host(ctx, hostID)
		if err != nil {
			o.renderRemediationPage(w, "Device Error", "Unable to retrieve device information.", nil)
			return nil
		}
	}

	// Check device health policies
	// For Phase 2: Check if device has any failing policies
	health, err := o.svc.ds.GetHostHealth(ctx, host.ID)
	if err != nil {
		// Cannot determine health, deny access
		o.renderRemediationPage(w, "Health Check Failed", "Unable to determine device health status.", nil)
		return nil
	}

	// If device has failing policies, get the details and show remediation
	if health.FailingPoliciesCount > 0 {
		policies, err := o.svc.ds.ListPoliciesForHost(ctx, host)
		if err != nil {
			o.renderRemediationPage(w, "Policy Check Failed", "Unable to retrieve policy information.", nil)
			return nil
		}

		// Filter to only failing policies
		var failingPolicies []*fleet.HostPolicy
		for _, policy := range policies {
			if policy.Response == "fail" {
				failingPolicies = append(failingPolicies, policy)
			}
		}

		o.renderRemediationPage(w, "Device health check failed",
			fmt.Sprintf("Your device has %d failing policy check(s). Please resolve the issues below:", len(failingPolicies)),
			failingPolicies)
		return nil
	}

	// Device is healthy, allow access
	return &saml.Session{
		NameID: email,
	}
}

func (o *oktaDeviceHealthSessionProvider) renderRemediationPage(w http.ResponseWriter, title, message string, failingPolicies []*fleet.HostPolicy) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Device Access Denied</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background-color: #F9FAFC;
        }
        .container {
            background-color: #FFFFFF;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            border: 1px solid #E2E4EA;
        }
        h1 {
            color: #D66C7B;
            margin-top: 0;
            font-size: 24px;
        }
        .message {
            font-size: 16px;
            color: #515774;
            margin-bottom: 30px;
            line-height: 1.5;
        }
        .policy {
            border-left: 4px solid #D66C7B;
            background-color: #F1F0FF;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .policy-name {
            font-weight: 600;
            font-size: 16px;
            color: #192147;
            margin-bottom: 8px;
        }
        .policy-description {
            color: #515774;
            margin-bottom: 10px;
            line-height: 1.5;
        }
        .policy-resolution {
            background-color: #FEF7E0;
            border-left: 4px solid #EBBC43;
            padding: 10px;
            margin-top: 10px;
            border-radius: 4px;
        }
        .resolution-title {
            font-weight: 600;
            color: #192147;
            margin-bottom: 5px;
        }
        .footer {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #E2E4EA;
            color: #8B8FA2;
            font-size: 14px;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>` + title + `</h1>
        <div class="message">` + message + `</div>`

	if len(failingPolicies) > 0 {
		for _, policy := range failingPolicies {
			html += `
        <div class="policy">
            <div class="policy-name">` + policy.Name + `</div>`

			if policy.Description != "" {
				html += `<div class="policy-description">` + policy.Description + `</div>`
			}

			if policy.Resolution != nil && *policy.Resolution != "" {
				html += `
            <div class="policy-resolution">
                <div class="resolution-title">How to resolve:</div>
                ` + *policy.Resolution + `
            </div>`
			}

			html += `</div>`
		}
	}

	html += `
        <div class="footer">
            Once you have resolved these issues, to to Fleet Desktop -> My device, refetch your device, and please try signing in again.
        </div>
    </div>
</body>
</html>`

	_, _ = w.Write([]byte(html))
}

// oktaDeviceHealthSPProvider implements saml.ServiceProviderProvider
type oktaDeviceHealthSPProvider struct{}

func (o *oktaDeviceHealthSPProvider) GetServiceProvider(r *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	// Return hardcoded Okta SP metadata
	// TODO: Look up from database by serviceProviderID
	return oktaServiceProviderMetadata, nil
}

// getOktaDeviceHealthIDP creates and configures the SAML IdP
func (s *Service) getOktaDeviceHealthIDP(baseMetadataURL, baseSSOURL string) (*saml.IdentityProvider, error) {
	// Parse metadata base URL
	metadataBase, err := url.Parse(baseMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseMetadataURL: %w", err)
	}

	metadataURL := *metadataBase
	metadataURL.Path = "/api/v1/fleet/okta/device_health/metadata"

	// Parse SSO base URL
	ssoBase, err := url.Parse(baseSSOURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseSSOURL: %w", err)
	}

	ssoURL := *ssoBase
	ssoURL.Path = "/api/v1/fleet/okta/device_health/sso"

	sessionProvider := &oktaDeviceHealthSessionProvider{svc: s}
	spProvider := &oktaDeviceHealthSPProvider{}

	// Use kitlog adapter to satisfy saml logger.Interface
	samlLogger := &kitlogAdapter{logger: kitlog.With(s.logger, "component", "saml-idp")}

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
