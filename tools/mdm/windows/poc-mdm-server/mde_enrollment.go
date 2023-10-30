package main

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EnrollHandler is the HTTP handler assosiated with the enrollment protocol's enrollment endpoint.
func EnrollHandler(w http.ResponseWriter, r *http.Request) {
	// Read The HTTP Request body
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	body := string(bodyRaw)

	// Retrieve the MessageID From The Body For The Response
	messageID := strings.Replace(strings.Replace(regexp.MustCompile(`<a:MessageID>[\s\S]*?<\/a:MessageID>`).FindStringSubmatch(body)[0], "<a:MessageID>", "", -1), "</a:MessageID>", "", -1)

	// Retrieve the BinarySecurityToken (which contains a Certificate Signing Request) From The Body For The Response
	binarySecurityToken := strings.Replace(strings.Replace(regexp.MustCompile(`<wsse:BinarySecurityToken ValueType="http:\/\/schemas.microsoft.com\/windows\/pki\/2009\/01\/enrollment#PKCS10" EncodingType="http:\/\/docs\.oasis-open\.org\/wss\/2004\/01\/oasis-200401-wss-wssecurity-secext-1\.0\.xsd#base64binary">[\s\S]*?<\/wsse:BinarySecurityToken>`).FindStringSubmatch(body)[0], `<wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/windows/pki/2009/01/enrollment#PKCS10" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">`, "", -1), "</wsse:BinarySecurityToken>", "", -1)

	// Retrieve the DeviceID From The Body For The Response
	deviceID := strings.Replace(strings.Replace(regexp.MustCompile(`<ac:ContextItem Name="DeviceID"><ac:Value>[\s\S]*?<\/ac:Value><\/ac:ContextItem>`).FindStringSubmatch(body)[0], `<ac:ContextItem Name="DeviceID"><ac:Value>`, "", -1), "</ac:Value></ac:ContextItem>", "", -1)

	// Retrieve the EnrollmentType From The Body For The Response
	enrollmentType := strings.Replace(strings.Replace(regexp.MustCompile(`<ac:ContextItem Name="EnrollmentType"><ac:Value>[\s\S]*?<\/ac:Value><\/ac:ContextItem>`).FindStringSubmatch(body)[0], `<ac:ContextItem Name="EnrollmentType"><ac:Value>`, "", -1), "</ac:Value></ac:ContextItem>", "", -1)

	/* Sign binary security token */
	// Load raw Root CA
	//rootCertificateDer, err := ioutil.ReadFile("./identity/identity.crt")
	rootCertificateDer, err := ioutil.ReadFile("./identity/dev_cert_mdmwindows_com.der")
	if err != nil {
		panic(err)
	}
	//rootPrivateKeyDer, err := ioutil.ReadFile("./identity/identity.key")
	rootPrivateKeyDer, err := ioutil.ReadFile("./identity/dev_cert_mdmwindows_com.key")
	if err != nil {
		panic(err)
	}

	// Convert the raw Root CA cert & key to parsed version
	rootCert, err := x509.ParseCertificate(rootCertificateDer)
	if err != nil {
		panic(err)
	}

	rootPrivateKey, err := x509.ParsePKCS1PrivateKey(rootPrivateKeyDer)
	if err != nil {
		panic(err)
	}

	// Decode Base64
	csrRaw, err := base64.StdEncoding.DecodeString(binarySecurityToken)
	if err != nil {
		panic(err)
	}

	// Decode and verify CSR
	//csr, err := x509.ParseCertificateRequest(csrRaw)
	csr, err := ParseCertificateRequest2(csrRaw)
	if err != nil {
		panic(err)
	}
	if err = csr.CheckSignature(); err != nil {
		panic(err)
	}

	// Create client identity certificate
	NotBefore1 := time.Now().Add(time.Duration(mathrand.Int31n(120)) * -time.Minute) // This randomises the creation time a bit for added security (Recommended by x509 signing article not the MDM spec)
	clientCertificate := &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,
		SerialNumber:       big.NewInt(2),
		Issuer:             rootCert.Issuer,
		Subject: pkix.Name{
			CommonName: deviceID,
		}, // The Subject is not used from the CSR because the characters in it are causing issues.
		NotBefore:   NotBefore1,
		NotAfter:    NotBefore1.Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Sign certificate with the identity
	clientCRTRaw, err := x509.CreateCertificate(rand.Reader, clientCertificate, rootCert, csr.PublicKey, rootPrivateKey)
	if err != nil {
		panic(err)
	}

	// Note: SHA-1 Hash OID is deprecated

	// Fingerprint (SHA-1 hash) of client certificate
	h := sha1.New()
	h.Write(clientCRTRaw)
	signedClientCertFingerprint := strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil))) // TODO: Cleanup -> This line is probally messer than it needs to be

	// Fingerprint (SHA-1 hash) of client certificate
	h2 := sha1.New()
	h2.Write(rootCertificateDer)
	identityCertFingerprint := strings.ToUpper(fmt.Sprintf("%x", h2.Sum(nil))) // TODO: Cleanup -> This line is probally messer than it needs to be

	// Determain Certstore
	certStore := "User"
	if enrollmentType == "Device" {
		certStore = "System"
	}

	// End Sign binary security token

	// Generate WAP provisioning profile for inside the payload
	wapProvisionProfile := `
		<?xml version="1.0" encoding="UTF-8"?>
		<wap-provisioningdoc version="1.1">
			<characteristic type="CertificateStore">
				<characteristic type="Root">
					<characteristic type="System">
						<characteristic type="` + identityCertFingerprint /* Root CA Certificate Fingureprint (SHA-1 hash of Der) */ + `">
							<parm name="EncodedCertificate" value="` + base64.StdEncoding.EncodeToString(rootCertificateDer) /* Base64 encoded root CA certificate */ + `" />
						</characteristic>
					</characteristic>
				</characteristic>
				<characteristic type="My">
					<characteristic type="` + certStore + `">
						<characteristic type="` + signedClientCertFingerprint /* Signed Client Certificate (From the BinarySecurityToken) Fingureprint (SHA-1 hash of Der) */ + `">
							<parm name="EncodedCertificate" value="` + base64.StdEncoding.EncodeToString(clientCRTRaw) /* Base64 encoded signed certificate */ + `" />
						</characteristic>
						<characteristic type="PrivateKeyContainer" />
					</characteristic>
					<characteristic type="WSTEP">
						<characteristic type="Renew">
							<parm name="ROBOSupport" value="true" datatype="boolean"/>
							<parm name="RenewPeriod" value="60" datatype="integer"/>
							<parm name="RetryInterval" value="4" datatype="integer"/>
						</characteristic>
					</characteristic>
				</characteristic>
			</characteristic>
			<characteristic type="APPLICATION">
				<parm name="APPID" value="w7" />
				<parm name="PROVIDER-ID" value="DEMO MDM" />
				<parm name="NAME" value="PoC Demo MDM Server" />
				<parm name="ADDR" value="https://` + domain + `/ManagementServer/MDM.svc" />
				<parm name="ServerList" value="https://` + domain + `/ManagementServer/ServerList.svc" />
				<parm name="ROLE" value="4294967295" />
				<parm name="BACKCOMPATRETRYDISABLED" />
				<parm name="USEHWDEVID" />				
				<parm name="CONNRETRYFREQ" value="6" />
				<parm name="INITIALBACKOFFTIME" value="30000" />
				<parm name="MAXBACKOFFTIME" value="120000" />
				<parm name="DEFAULTENCODING" value="application/vnd.syncml.dm+xml" />
				<characteristic type="APPAUTH">
					<parm name="AAUTHLEVEL" value="CLIENT"/>
					<parm name="AAUTHTYPE" value="DIGEST"/>					
					<parm name="AAUTHSECRET" value="2jsidqgffx"/>
					<parm name="AAUTHDATA" value="MzA5Mzc5MTU4MQ=="/>
				</characteristic>
				<characteristic type="APPAUTH">
					<parm name="AAUTHLEVEL" value="APPSRV"/>
					<parm name="AAUTHTYPE" value="DIGEST"/>
					<parm name="AAUTHNAME" value="43f8bf59-75dd-4849-9e0f-9b8557346034"/>
					<parm name="AAUTHSECRET" value="wrer3w5csb"/>
					<parm name="AAUTHDATA" value="MzA5Mzc5MTU4MQ=="/>
				</characteristic>
			</characteristic>
			<characteristic type="DMClient">
				<characteristic type="Provider">
					<characteristic type="DEMO MDM">
						<parm name="UPN" value="infected@mdmwindows.com" />	
						<parm name="EnableOmaDmKeepAliveMessage" value="true" datatype="boolean" />						
						<parm name="RequireMessageSigning" value="true" datatype="boolean" />
						<characteristic type="Poll">
							<parm name="NumberOfFirstRetries" value="0" datatype="integer" />
							<parm name="IntervalForFirstSetOfRetries" value="1" datatype="integer" />
							<parm name="NumberOfSecondRetries" value="0" datatype="integer" />
							<parm name="IntervalForSecondSetOfRetries" value="1" datatype="integer" />
							<parm name="NumberOfRemainingScheduledRetries" value="0" datatype="integer" />
							<parm name="IntervalForRemainingScheduledRetries" value="1560" datatype="integer" />
							<parm name="PollOnLogin" value="true" datatype="boolean" />
						</characteristic>
					</characteristic>
				</characteristic>
			</characteristic>	
		</wap-provisioningdoc>`

	wapProvisionProfileRaw := []byte(strings.ReplaceAll(strings.ReplaceAll(wapProvisionProfile, "\n", ""), "\t", ""))

	fmt.Printf("======================================\n%s\n======================================\n", string(wapProvisionProfileRaw))

	response := []byte(` 
		<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
		  <s:Header>
			<Action mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RSTRC/wstep</Action>
			<a:RelatesTo>` + messageID + `</a:RelatesTo>
			<o:Security xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" s:mustUnderstand="1">
			  <u:Timestamp u:Id="_0">
				<u:Created>2023-06-14T17:34:39.314Z</u:Created>
				<u:Expires>2023-06-14T17:44:39.314Z</u:Expires>
			  </u:Timestamp>
			</o:Security>
		  </s:Header>
		  <s:Body>
			<RequestSecurityTokenResponseCollection xmlns="http://docs.oasis-open.org/ws-sx/ws-trust/200512">
			  <RequestSecurityTokenResponse>
				<TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</TokenType>
				<DispositionMessage xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollment"></DispositionMessage>
				<RequestedSecurityToken>
				  <BinarySecurityToken xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" ValueType="http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentProvisionDoc" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + base64.StdEncoding.EncodeToString(wapProvisionProfileRaw) + `</BinarySecurityToken>
				</RequestedSecurityToken>
				<RequestID xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollment">0</RequestID>
			  </RequestSecurityTokenResponse>
			</RequestSecurityTokenResponseCollection>
		  </s:Body>
		</s:Envelope>`)

	// Return response body
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Write(response)
}
