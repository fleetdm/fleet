package mdmtest

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testhelpers"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/kitlogadapter"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/google/uuid"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/smallstep/scep"
	"golang.org/x/crypto/acme"
)

// TestAppleMDMClient simulates a macOS MDM client.
type TestAppleMDMClient struct {
	// UUID is a random fake unique ID of the simulated device.
	UUID string
	// SerialNumber is a random fake serial number of the simulated device.
	SerialNumber string
	// Model is the model of the simulated device.
	Model string
	// OSVersion is the version of the operating system of the simulated device.
	OSVersion string

	// EnrollInfo holds the information necessary to enroll to an MDM server.
	EnrollInfo AppleEnrollInfo

	// UserUUID is a random fake unique ID of a simulated user. Only filled in if a user enrollment
	// is done
	UserUUID string
	// Username is the username of a simulated user. Only filled in if a user enrollment is done
	Username string

	// SecretUUID is a random fake unique ID of a simulated device used for providing consistent
	// identifiers for enrollments like Apple's Account Driven User Enrollment in which the device
	// never actually provides a UDID to the server, instead generating a random one for each
	// enrollment
	secretUUID string

	// fleetServerURL is the URL of the Fleet server, used to fetch the enrollment profile.
	fleetServerURL string

	// debug enables debug logging of request/responses.
	debug bool

	// fetchEnrollmentProfileFromDesktop indicates whether this simulated device
	// will fetch the enrollment profile from Fleet as if it were a device running
	// Fleet Desktop.
	fetchEnrollmentProfileFromDesktop bool
	// desktopURLToken is the Fleet Desktop token used to fetch the enrollment profile
	// from Fleet as if it were a device running Fleet Desktop.
	desktopURLToken string

	// fetchEnrollmentProfileFromDEP indicates whether this simulated device will fetch
	// the enrollment profile from Fleet as if it were a device running the DEP flow.
	fetchEnrollmentProfileFromDEP bool
	// fetchEnrollmentProfileFromDEPUsingPost functions the same as fetchEnrollmentProfileFromDEP
	// except that it uses a POST request instead of a GET request.
	fetchEnrollmentProfileFromDEPUsingPost bool

	// fetchEnrollmentProfileFromOTA indicates whether this simulated device will fetch
	// the enrollment profile from Fleet as if it were a device running the OTA flow.
	fetchEnrollmentProfileFromOTA bool
	// otaEnrollSecret is the team enroll secret to be used during the OTA flow.
	otaEnrollSecret string
	// otaIdpUUID is the optional uuid of the idp account that should be associated with the host enrolling
	otaIdpUUID string

	// fetchEnrollmentProfileFromMDMBYOD indicates whether this simulated device will fetch
	// the enrollment profile from Fleet as if it were a device running the Account Driven User
	// Enrollment flow sometimes called MDM-BYOD in Apple docs.
	fetchEnrollmentProfileFromMDMBYOD bool

	// The bearer token, if set and fetchEnrollmentProfileFromMDMBYOD is set, will be passed in the
	// Authorization header on all MDM requests as discussed in Ongoing authorization [1] in the
	// Apple account driven enrollment docs.
	//
	// [1] https://developer.apple.com/documentation/devicemanagement/onboarding-users-with-account-sign-in#overview
	authorizationBearerToken string

	// desktopURLToken is the token used to fetch the enrollment profile
	// from Fleet as if it were a device running the DEP flow.
	depURLToken string

	// scepCert contains the SCEP client certificate generated during the
	// SCEP enrollment process.
	scepCert *x509.Certificate
	// scepKey contains the SCEP client private key generated during the
	// SCEP enrollment process.
	scepKey *rsa.PrivateKey

	acmeCertCA    *x509.Certificate
	acmeCertCAKey *ecdsa.PrivateKey

	acmeCert *x509.Certificate
	acmeKey  *ecdsa.PrivateKey

	// ACME client used to enroll via ACME if enrollment profile is ACME based.
	acmeClient *acme.Client

	// legacyIDeviceEnrollRef is an optional enroll reference that will be added to the MDMURL after the
	// client fetches the enrollment profile but prior to attempting SCEP enrollment. Note that this
	// is not a full simulation of legacy enrollments (especially, as it related to IdP). Rather it
	// is enough to test certain SCEP renewal scenarios for iOS/IPadOS devices
	legacyIDeviceEnrollRef string

	// skipParseEnrollProf, when set to true, will skip parsing the enrollment profile after
	// fetching it. Instead, the raw profile bytes will still be stored in enrollProfBytes.
	skipParseEnrollProf bool
}

// TestMDMAppleClientOption allows configuring a TestMDMClient.
type TestMDMAppleClientOption func(*TestAppleMDMClient)

// TestMDMAppleClientDebug configures the TestMDMClient to run in debug mode.
func TestMDMAppleClientDebug() TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.debug = true
	}
}

func WithEnrollmentProfileFromDEPUsingPost() TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.fetchEnrollmentProfileFromDEPUsingPost = true
		c.fetchEnrollmentProfileFromDEP = false
	}
}

// Will set a cookie for OTA requests which mimics SSO being enabled before OTA enrollment.
func WithOTAIdpUUID(idpUUID string) TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.otaIdpUUID = idpUUID
	}
}

func WithSkipParseEnrollProf(skip bool) TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.skipParseEnrollProf = skip
	}
}

// Will set ACME CA certs, which is required if the device enrolls via the ACME flow
func WithACMECerts(certCA *x509.Certificate, certKey *ecdsa.PrivateKey) TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.acmeCertCA = certCA
		c.acmeCertCAKey = certKey
	}
}

// Will add the specified reference as a query parameter to the MDMURL after the
// client fetches the enrollment profile but prior to attempting SCEP enrollment. Note that this
// is not a full simulation of legacy enrollments (especially, as it relates to IdP). Rather it
// is enough to test certain SCEP renewal scenarios for iOS/IPadOS devices
func WithLegacyIDeviceEnrollRef(enrollRef string) TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.legacyIDeviceEnrollRef = enrollRef
	}
}

// AppleEnrollInfo contains the necessary information to enroll to an MDM server.
type AppleEnrollInfo struct {
	// SCEPChallenge is the SCEP challenge to present to the SCEP server when enrolling.
	SCEPChallenge string
	// SCEPURL is the URL of the SCEP server.
	SCEPURL string
	// MDMURL is the URL of the MDM server.
	MDMURL string
	// AssignedManagedAppleID is the Assigned Managed Apple account for the device. Only used for
	// account driven enrollment flows, so it will not always be available.
	AssignedManagedAppleID string
	// ACMEURL is the optional URL that will be used for ACME enrollment instead of the SCEP.
	// Currently, this is only used for certain enrollment scenarios when
	// config.mdm.apple_require_hardware_attestation is true.
	ACMEURL string

	// RawProfile contains the raw bytes of the enrollment profile. This is useful for tests that
	// want to inspect the actual profile content. This field is populated regardless of the value
	// of skipParseEnrollProf.
	RawProfile []byte
}

// NewTestMDMClientAppleDesktopManual will create a simulated device that will fetch
// enrollment profile from Fleet as if it were a device running Fleet Desktop.
func NewTestMDMClientAppleDesktopManual(serverURL string, desktopURLToken string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:         strings.ToUpper(uuid.New().String()),
		SerialNumber: RandSerialNumber(),
		Model:        "MacBookPro16,1",

		fetchEnrollmentProfileFromDesktop: true,
		desktopURLToken:                   desktopURLToken,

		fleetServerURL: serverURL,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

// NewTestMDMClientAppleDEP will create a simulated device that will fetch
// enrollment profile from Fleet as if it were a device running the DEP flow.
func NewTestMDMClientAppleDEP(serverURL string, depURLToken string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:         strings.ToUpper(uuid.New().String()),
		SerialNumber: RandSerialNumber(),
		Model:        "MacBookPro16,1",

		fetchEnrollmentProfileFromDEP: true,
		depURLToken:                   depURLToken,

		fleetServerURL: serverURL,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

// NewTestMDMClientAppleDEPFromDevice will create a simulated device that will fetch
// enrollment profile from Fleet as if it were a device running the DEP flow.
// The deviceSerialNumber is used as the serial number of the device
// The model is used as the model of the device
func NewTestMDMClientAppleDEPFromDevice(serverURL string, depURLToken string, deviceSerialNumber string, model string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:         strings.ToUpper(uuid.New().String()),
		SerialNumber: deviceSerialNumber,
		Model:        model,

		fetchEnrollmentProfileFromDEPUsingPost: true,
		depURLToken:                            depURLToken,

		fleetServerURL: serverURL,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

// NewTestMDMClientAppleDirect will create a simulated device that will not fetch the enrollment
// profile from Fleet. The enrollment information is to be provided in the enrollInfo.
func NewTestMDMClientAppleDirect(enrollInfo AppleEnrollInfo, model string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:         strings.ToUpper(uuid.New().String()),
		SerialNumber: RandSerialNumber(),
		Model:        model,

		EnrollInfo: enrollInfo,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

// NewTestMDMClientAppleOTA will create a simulated device that will fetch
// enrollment profile from Fleet as if it were a device running the Over The
// Air (OTA) flow.
func NewTestMDMClientAppleOTA(serverURL, enrollSecret, model string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:                          strings.ToUpper(uuid.New().String()),
		SerialNumber:                  RandSerialNumber(),
		Model:                         model,
		fetchEnrollmentProfileFromOTA: true,
		fleetServerURL:                serverURL,
		otaEnrollSecret:               enrollSecret,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

func NewTestMDMClientAppleAccountDrivenUserEnrollment(serverURL, model, authorizationBearerToken string, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	// NB An Account Driven User Enrollment has no actual UDID or serial but does have a randomly
	// generated enrollment ID
	c := TestAppleMDMClient{
		Model:                             model,
		fetchEnrollmentProfileFromMDMBYOD: true,
		authorizationBearerToken:          authorizationBearerToken,
		fleetServerURL:                    serverURL,
	}
	c.secretUUID = strings.ToUpper(uuid.New().String())
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

func (c *TestAppleMDMClient) Identifier() string {
	if c.UUID != "" {
		return c.UUID
	}
	return c.secretUUID
}

func (c *TestAppleMDMClient) EnrollmentID() string {
	return "testenrollmentid-" + c.Identifier()
}

func (c *TestAppleMDMClient) SetDesktopToken(tok string) {
	c.desktopURLToken = tok
}

func (c *TestAppleMDMClient) SetDEPToken(tok string) {
	c.depURLToken = tok
}

// Enroll runs the MDM enroll protocol on the simulated device. It fetches the enrollment
// profile from the Fleet server and then runs the SCEP enrollment, Authenticate and TokenUpdate
// steps.
func (c *TestAppleMDMClient) Enroll() error {
	return c.enrollDevice(true)
}

func (c *TestAppleMDMClient) enrollDevice(awaitingConfiguration bool) error {
	switch {
	case c.fetchEnrollmentProfileFromDesktop:
		if err := c.fetchEnrollmentProfileFromDesktopURL(); err != nil {
			return fmt.Errorf("get enrollment profile from desktop URL: %w", err)
		}
	case c.fetchEnrollmentProfileFromDEPUsingPost:
		if err := c.fetchEnrollmentProfileFromDEPURLUsingPost(); err != nil {
			return fmt.Errorf("get enrollment profile using POST from DEP URL: %w", err)
		}
	case c.fetchEnrollmentProfileFromDEP:
		if err := c.fetchEnrollmentProfileFromDEPURL(); err != nil {
			return fmt.Errorf("get enrollment profile from DEP URL: %w", err)
		}
	case c.fetchEnrollmentProfileFromOTA:
		if err := c.fetchEnrollmentProfileFromOTAURL(); err != nil {
			return fmt.Errorf("get enrollment profile from OTA URL: %w", err)
		}
	case c.fetchEnrollmentProfileFromMDMBYOD:
		if err := c.fetchEnrollmentProfileFromMDMBYODURL(); err != nil {
			return fmt.Errorf("get enrollment profile from MDM BYOD URL: %w", err)
		}
	default:
		if c.EnrollInfo.SCEPURL == "" || c.EnrollInfo.MDMURL == "" || c.EnrollInfo.SCEPChallenge == "" {
			return fmt.Errorf("missing info needed to perform enrollment: %+v", c.EnrollInfo)
		}
	}

	if c.legacyIDeviceEnrollRef != "" {
		parsedMDMURL, err := url.Parse(c.EnrollInfo.MDMURL)
		if err != nil {
			return fmt.Errorf("parsing MDM URL: %w", err)
		}
		q := parsedMDMURL.Query()
		q.Set("enroll_reference", c.legacyIDeviceEnrollRef)
		parsedMDMURL.RawQuery = q.Encode()
		c.EnrollInfo.MDMURL = parsedMDMURL.String()
	}

	if c.acmeClient != nil {
		// Do ACME enrollment
		if err := c.ACMEEnroll(); err != nil {
			return fmt.Errorf("ACME enroll: %w", err)
		}
	} else {
		if err := c.SCEPEnroll(); err != nil {
			return fmt.Errorf("scep enroll: %w", err)
		}
	}
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	if err := c.TokenUpdate(awaitingConfiguration); err != nil {
		return fmt.Errorf("token update: %w", err)
	}
	return nil
}

// Re-enroll runs the MDM enroll protocol on the simulated device, but TokenUpdate is not set as AwaitingConfiguration
func (c *TestAppleMDMClient) Reenroll() error {
	return c.enrollDevice(false)
}

func (c *TestAppleMDMClient) UserEnroll() error {
	c.UserUUID = strings.ToUpper(uuid.New().String())
	c.Username = "fleetie" + randStr(5)
	return c.UserTokenUpdate()
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromDesktopURL() error {
	request, err := http.NewRequest("GET", c.fleetServerURL+"/api/latest/fleet/device/"+c.desktopURLToken+"/mdm/apple/manual_enrollment_profile", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	// #nosec (this client is used for testing only)
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}))

	response, err := cc.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var dest struct {
		EnrollURL string `json:"enroll_url,omitempty"`
	}
	err = json.Unmarshal(body, &dest)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if dest.EnrollURL == "" {
		return errors.New("empty enroll URL in response")
	}

	urlParsed, err := url.Parse(dest.EnrollURL)
	if err != nil {
		return fmt.Errorf("parse enroll URL: %w", err)
	}
	q := urlParsed.Query()
	es := q.Get("enroll_secret")
	if es == "" {
		return errors.New("missing enroll_secret in enroll URL")
	}
	c.otaEnrollSecret = es

	return c.fetchEnrollmentProfileFromOTAURL()
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromDEPURL() error {
	di, err := EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial:    c.SerialNumber,
		UDID:      c.UUID,
		Product:   c.Model,
		OSVersion: c.OSVersion,
	})
	if err != nil {
		return fmt.Errorf("test client: encoding device info: %w", err)
	}
	return c.fetchEnrollmentProfile(
		apple_mdm.EnrollPath+"?token="+c.depURLToken+"&deviceinfo="+di, nil,
	)
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromDEPURLUsingPost() error {
	buf, err := MachineInfoAsPKCS7(fleet.MDMAppleMachineInfo{
		Serial:    c.SerialNumber,
		UDID:      c.UUID,
		Product:   c.Model,
		OSVersion: c.OSVersion,
	})
	if err != nil {
		return fmt.Errorf("test client: encoding device info: %w", err)
	}
	return c.fetchEnrollmentProfile(
		apple_mdm.EnrollPath+"?token="+c.depURLToken, buf,
	)
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromOTAURL() error {
	return c.fetchOTAProfile(
		"/api/latest/fleet/enrollment_profiles/ota?enroll_secret=" + url.QueryEscape(c.otaEnrollSecret),
	)
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromMDMBYODURL() error {
	buf, err := AccountDrivenUserEnrollDeviceInfoAsPKCS7(fleet.MDMAppleAccountDrivenUserEnrollDeviceInfo{
		Product:  c.Model,
		Version:  "22A3351", // iOS 18.0, but it doesn't really matter
		Language: "en-US",
	})
	if err != nil {
		return fmt.Errorf("test client: encoding device info: %w", err)
	}
	return c.fetchEnrollmentProfile(
		apple_mdm.AccountDrivenEnrollPath, buf,
	)
}

func (c *TestAppleMDMClient) fetchOTAProfile(url string) error {
	request, err := http.NewRequest("GET", c.fleetServerURL+url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	// #nosec (this client is used for testing only)
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}))

	if c.otaIdpUUID != "" {
		request.AddCookie(&http.Cookie{
			Name:  shared_mdm.BYODIdpCookieName,
			Value: c.otaIdpUUID,
		})
	}

	response, err := cc.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	p7, err := pkcs7.Parse(body)
	if err != nil {
		return fmt.Errorf("OTA profile is not XML nor PKCS7 parseable: %w", err)
	}
	err = p7.Verify()
	if err != nil {
		return fmt.Errorf("verifying OTA profile: %w", err)
	}

	var otaEnrollmentProfile struct {
		PayloadContent struct {
			URL string `plist:"URL"`
		} `plist:"PayloadContent"`
	}
	err = plist.Unmarshal(p7.Content, &otaEnrollmentProfile)
	if err != nil {
		return fmt.Errorf("unmarshaling OTA enrollment response: %w", err)
	}

	rawDeviceInfo := []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PRODUCT</key>
	<string>%s</string>
	<key>SERIAL</key>
	<string>%s</string>
	<key>UDID</key>
	<string>%s</string>
	<key>VERSION</key>
	<string>22A5316k</string>
</dict>
</plist>`, c.Model, c.SerialNumber, c.UUID))

	do := func(cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
		signedData, err := pkcs7.NewSignedData(rawDeviceInfo)
		if err != nil {
			return nil, fmt.Errorf("create signed data: %w", err)
		}
		err = signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{})
		if err != nil {
			return nil, fmt.Errorf("add signer: %w", err)
		}
		sig, err := signedData.Finish()
		if err != nil {
			return nil, fmt.Errorf("finish signing: %w", err)
		}

		request, err := http.NewRequest(
			"POST",
			otaEnrollmentProfile.PayloadContent.URL,
			bytes.NewReader(sig),
		)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		// #nosec (this client is used for testing only)
		cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		response, err := cc.Do(request)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		return body, nil
	}

	// TODO(roberto 09-10-2024): the first request in the OTA flow must be
	// signed using a keypair that has a valid Apple certificate as root. I
	// believe this could be done with a little bit of reverse
	// engineering/cleverness but for now, we're signing the request with
	// our mock certs and setting this env var to skip the verification.

	mockedCert, mockedKey, err := apple_mdm.NewSCEPCACertKey()
	if err != nil {
		return fmt.Errorf("creating mock certificates: %w", err)
	}
	dev_mode.SetOverride("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY", "1")
	body, err = do(mockedCert, mockedKey)
	dev_mode.ClearOverride("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY")
	if err != nil {
		return fmt.Errorf("first OTA request: %w", err)
	}

	var scepInfo struct {
		PayloadContent []struct {
			PayloadContent struct {
				Challenge string `plist:"Challenge"`
				URL       string `plist:"URL"`
			} `plist:"PayloadContent"`
		} `plist:"PayloadContent"`
	}

	err = plist.Unmarshal(body, &scepInfo)
	if err != nil {
		return fmt.Errorf("unmarshaling SCEP response: %w", err)
	}

	tmpCert, tmpKey, err := c.doSCEP(scepInfo.PayloadContent[0].PayloadContent.URL, scepInfo.PayloadContent[0].PayloadContent.Challenge)
	if err != nil {
		return fmt.Errorf("get SCEP certificate for OTA: %w", err)
	}

	body, err = do(tmpCert, tmpKey)
	if err != nil {
		return fmt.Errorf("seconde OTA request: %w", err)
	}
	p7, err = pkcs7.Parse(body)
	if err != nil {
		return fmt.Errorf("enrollment profile is not XML nor PKCS7 parseable: %w", err)
	}
	err = p7.Verify()
	if err != nil {
		return fmt.Errorf("verifying enrollment profile: %w", err)
	}
	if c.skipParseEnrollProf {
		c.EnrollInfo.RawProfile = p7.Content
		return nil
	}
	enrollInfo, err := ParseEnrollmentProfile(p7.Content)
	if err != nil {
		return fmt.Errorf("parse OTA SCEP profile: %w", err)
	}
	enrollInfo.RawProfile = p7.Content
	c.EnrollInfo = *enrollInfo
	return nil
}

func (c *TestAppleMDMClient) fetchEnrollmentProfile(path string, body []byte) (err error) {
	var request *http.Request
	if len(body) > 0 {
		request, err = http.NewRequest("POST", c.fleetServerURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		request.Header.Set("Content-Type", "application/pkcs7-signature")
	} else {
		request, err = http.NewRequest("GET", c.fleetServerURL+path, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
	}
	if c.fetchEnrollmentProfileFromMDMBYOD && c.authorizationBearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.authorizationBearerToken)
	}
	// #nosec (this client is used for testing only)
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}))
	response, err := cc.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}
	rspBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if err := response.Body.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}

	rawProfile := rspBody
	if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
		p7, err := pkcs7.Parse(rspBody)
		if err != nil {
			return fmt.Errorf("enrollment profile is not XML nor PKCS7 parseable: %w", err)
		}

		err = p7.Verify()
		if err != nil {
			return err
		}

		rawProfile = p7.Content
	}
	if c.skipParseEnrollProf {
		c.EnrollInfo.RawProfile = rawProfile
		return nil
	}
	enrollInfo, err := ParseEnrollmentProfile(rawProfile)
	if err != nil {
		return fmt.Errorf("parse enrollment profile: %w", err)
	}
	enrollInfo.RawProfile = rawProfile
	c.EnrollInfo = *enrollInfo

	if enrollInfo.ACMEURL != "" {
		if c.acmeCertCA == nil || c.acmeCertCAKey == nil {
			return errors.New("ACME enrollment requested but no cert/key provided")
		}
		c.acmeClient = &acme.Client{
			Key:          c.acmeCertCAKey,
			DirectoryURL: enrollInfo.ACMEURL,
			HTTPClient:   fleethttp.NewClient(),
		}
	}

	return nil
}

func (c *TestAppleMDMClient) doSCEP(url, challenge string) (*x509.Certificate, *rsa.PrivateKey, error) {
	ctx := context.Background()

	var logger *slog.Logger
	if c.debug {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	} else {
		logger = slog.New(slog.DiscardHandler)
	}
	client, err := newSCEPClient(url, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("scep client: %w", err)
	}

	// (1). Get the CA certificate from the SCEP server.
	resp, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("get CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificates(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA cert: %w", err)
	}

	// (2). Generate RSA key pair.
	devicePrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate RSA private key: %w", err)
	}

	// (3). Generate CSR.
	cn := fmt.Sprintf("fleet-testdevice-%s", c.Identifier())
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   cn,
				Organization: []string{"fleet-organization"},
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: challenge,
	}
	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, devicePrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create CSR: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CSR: %w", err)
	}

	// (4). SCEP requires a certificate for client authentication. We generate a new one
	// that uses the same CommonName and Key that we are trying to have signed.
	//
	// From RFC-8894:
	// If the client does not have an appropriate existing certificate, then a locally generated
	// self-signed certificate MUST be used. The keyUsage extension in the certificate MUST indicate that
	// it is valid for digitalSignature and keyEncipherment (if available). The self-signed certificate
	// SHOULD use the same subject name and key as in the PKCS #10 request.
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	certSerialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("generate cert serial number: %w", err)
	}
	deviceCertificateTemplate := x509.Certificate{
		SerialNumber: certSerialNumber,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: csr.Subject.Organization,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	deviceCertificateDerBytes, err := x509.CreateCertificate(
		rand.Reader,
		&deviceCertificateTemplate,
		&deviceCertificateTemplate,
		&devicePrivateKey.PublicKey,
		devicePrivateKey,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create device certificate: %w", err)
	}
	deviceCertificateForRequest, err := x509.ParseCertificate(deviceCertificateDerBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse device certificate: %w", err)
	}

	// (5). Send the PKCSReq message to the SCEP server.
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCert,
		SignerKey:   devicePrivateKey,
		SignerCert:  deviceCertificateForRequest,
		CSRReqMessage: &scep.CSRReqMessage{
			ChallengePassword: c.EnrollInfo.SCEPChallenge,
		},
	}
	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(kitlogadapter.NewLogger(logger)))
	if err != nil {
		return nil, nil, fmt.Errorf("create CSR request: %w", err)
	}
	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf("do CSR request: %w", err)
	}
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(kitlogadapter.NewLogger(logger)), scep.WithCACerts(msg.Recipients))
	if err != nil {
		return nil, nil, fmt.Errorf("parse PKIMessage response: %w", err)
	}
	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		return nil, nil, fmt.Errorf("PKIMessage CSR request failed with code: %s, fail info: %s", pkiMsgResp.PKIStatus, pkiMsgResp.FailInfo)
	}
	if err := pkiMsgResp.DecryptPKIEnvelope(deviceCertificateForRequest, devicePrivateKey); err != nil {
		return nil, nil, fmt.Errorf("decrypt PKI envelope: %w", err)
	}

	if c.debug {
		fmt.Println("SCEP enrollment successful")
	}

	// (6). return the signed certificate returned from the server as the device certificate and key.
	return pkiMsgResp.CertRepMessage.Certificate, devicePrivateKey, nil
}

// SCEPEnroll runs the SCEP enroll protocol for the simulated device.
func (c *TestAppleMDMClient) SCEPEnroll() error {
	cert, key, err := c.doSCEP(c.EnrollInfo.SCEPURL, c.EnrollInfo.SCEPChallenge)
	if err != nil {
		return err
	}

	c.scepCert = cert
	c.scepKey = key
	return nil
}

func (c *TestAppleMDMClient) ACMEEnroll() error {
	if c.acmeClient == nil {
		return errors.New("ACME URL not set in enrollment profile")
	}
	ctx := context.Background()
	_, err := c.acmeClient.Register(ctx, &acme.Account{}, func(tosURL string) bool { return true })
	if err != nil {
		return fmt.Errorf("ACME register account: %w", err)
	}

	order, err := c.acmeClient.AuthorizeOrder(ctx, []acme.AuthzID{{Type: "permanent-identifier", Value: c.SerialNumber}})
	if err != nil {
		return fmt.Errorf("ACME authorize order: %w", err)
	}

	if len(order.AuthzURLs) != 1 {
		// We only create on authz for an order
		return fmt.Errorf("expected 1 authz URL, got %d", len(order.AuthzURLs))
	}

	authz, err := c.acmeClient.GetAuthorization(ctx, order.AuthzURLs[0])
	if err != nil {
		return fmt.Errorf("ACME get authorization: %w", err)
	}

	if len(authz.Challenges) != 1 {
		// We only create one challenge for an authz
		return fmt.Errorf("expected 1 challenge, got %d", len(authz.Challenges))
	}

	if authz.Challenges[0].Type != "device-attest-01" {
		return fmt.Errorf("expected challenge type device-attest-01, got %s", authz.Challenges[0].Type)
	}

	challenge := authz.Challenges[0]
	leafCert, err := testhelpers.BuildAttestationLeafCert(c.acmeCertCA, c.acmeCertCAKey, c.SerialNumber, challenge.Token)
	if err != nil {
		return fmt.Errorf("build attestation leaf cert: %w", err)
	}
	payload, err := testhelpers.BuildAppleDeviceAttestationPayload(leafCert, c.acmeCertCA)
	if err != nil {
		return fmt.Errorf("build Apple device attestation payload: %w", err)
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	challenge.Payload = jsonPayload
	challenge, err = c.acmeClient.Accept(ctx, challenge)
	if err != nil {
		return fmt.Errorf("ACME accept challenge: %w", err)
	}

	if challenge.Status != "valid" {
		return fmt.Errorf("challenge not valid after acceptance, status: %s", challenge.Status)
	}

	encoded, acmeKey, err := testhelpers.GenerateCSRDER(c.SerialNumber)
	if err != nil {
		return fmt.Errorf("generate CSR DER: %w", err)
	}

	// crypto/acme lib base64encodes inside the method, so we have to decode it again here
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode CSR DER: %w", err)
	}

	der, _, err := c.acmeClient.CreateOrderCert(ctx, order.FinalizeURL, decoded, false)
	if err != nil {
		return fmt.Errorf("ACME create order cert and fetch cert: %w", err)
	}

	if len(der) != 1 {
		// Since we don't bundle in CreateOrderCert, we only expect the leaf cert that we can sign requests with.
		return fmt.Errorf("expected 1 certificate in ACME response, got %d", len(der))
	}

	acmeCert, err := x509.ParseCertificate(der[0])
	if err != nil {
		return fmt.Errorf("parse x509 ACME certificate: %w", err)
	}

	c.acmeCert = acmeCert
	// We can reuse the same key we used for the CSR since it's the one that matches the cert
	c.acmeKey = acmeKey

	return nil
}

// Authenticate sends the Authenticate message to the MDM server (Check In protocol).
func (c *TestAppleMDMClient) Authenticate() error {
	payload := map[string]any{
		"MessageType":  "Authenticate",
		"Model":        c.Model,
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
	}
	if !c.fetchEnrollmentProfileFromMDMBYOD {
		payload["UDID"] = c.UUID
		payload["SerialNumber"] = c.SerialNumber
		payload["DeviceName"] = "testdevice" + c.SerialNumber
	}
	if strings.HasPrefix(c.Model, "iPhone") || strings.HasPrefix(c.Model, "iPod") || strings.HasPrefix(c.Model, "iPad") {
		payload["ProductName"] = c.Model
	}
	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// TokenUpdate sends the TokenUpdate message to the MDM server (Check In protocol).
func (c *TestAppleMDMClient) TokenUpdate(awaitingConfiguration bool) error {
	pushMagic := "pushmagic" + c.SerialNumber
	token := []byte("token" + c.SerialNumber)
	unlockToken := []byte("unlocktoken" + c.SerialNumber)
	if c.SerialNumber == "" {
		pushMagic = "pushmagic" + c.Identifier()
		token = []byte("token" + c.Identifier())
		unlockToken = []byte("unlocktoken" + c.Identifier())
	}
	payload := map[string]any{
		"MessageType":  "TokenUpdate",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"NotOnConsole": "false",
		"PushMagic":    pushMagic,
		"Token":        token,
		"UnlockToken":  unlockToken,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	if awaitingConfiguration {
		payload["AwaitingConfiguration"] = true
	}
	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// TokenUpdate sends the TokenUpdate message with a username to the MDM server (Check In protocol).
// This creates a user channel pushtoken and an Enrollment with Type=User in nanomdm.
func (c *TestAppleMDMClient) UserTokenUpdate() error {
	if c.UserUUID == "" || c.Username == "" {
		return errors.New("user UUID and username must be set for user enrollment")
	}
	pushMagic := "pushmagic.user." + c.SerialNumber
	token := []byte("token.user." + c.SerialNumber)
	if c.SerialNumber == "" {
		pushMagic = "pushmagic.user." + c.Identifier()
		token = []byte("token.user." + c.Identifier())
	}
	payload := map[string]any{
		"MessageType":   "TokenUpdate",
		"Topic":         "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID":  "testenrollmentid-" + c.Identifier(),
		"NotOnConsole":  "false",
		"PushMagic":     pushMagic,
		"Token":         token,
		"UserID":        c.UserUUID,
		"UserLongName":  c.Username,
		"UserShortName": c.Username,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}

	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// UserAuthenticate sends the UserAuthenticate message to the MDM server (Check In protocol).
// Note that this is separate from the UserTokenUpdate and Authenticate used in device+user enrollment above.
// Fleet does not currently support UserAuthenticate so this is stubbed out just to test the HTTP error
//
// For more details see https://developer.apple.com/documentation/devicemanagement/userauthenticaterequest
func (c *TestAppleMDMClient) UserAuthenticate() error {
	if c.UUID == "" {
		return errors.New("UUID must be set for UserAuthenticate")
	}
	payload := map[string]any{
		"MessageType": "UserAuthenticate",
		"UDID":        c.UUID,
		"UserID":      uuid.New().String(),
	}

	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// DeclarativeManagement sends a DeclarativeManagement checkin request to the server.
//
// The endpoint argument is used as the value for the `Endpoint` key in the request payload.
//
// For more details check https://developer.apple.com/documentation/devicemanagement/declarativemanagementrequest
func (c *TestAppleMDMClient) DeclarativeManagement(endpoint string, data ...fleet.MDMAppleDDMStatusReport) (*http.Response, error) {
	payload := map[string]any{
		"MessageType":  "DeclarativeManagement",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"Endpoint":     endpoint,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	if len(data) != 0 {
		rawData, err := json.Marshal(data[0])
		if err != nil {
			return nil, fmt.Errorf("marshaling status report: %w", err)
		}
		payload["Data"] = rawData
	}
	r, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return r, err
}

// Checkout sends the CheckOut message to the MDM server.
func (c *TestAppleMDMClient) Checkout() error {
	payload := map[string]any{
		"MessageType":  "CheckOut",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// Idle sends an Idle message to the MDM server.
//
// Devices send an Idle status to signal the server that they're ready to
// receive commands. The server can signal back with either a command to run
// or an empty (nil, nil) response body to end the communication
// (i.e. no commands to run).
func (c *TestAppleMDMClient) Idle() (*mdm.Command, error) {
	payload := map[string]any{
		"Status":       "Idle",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	return c.sendAndDecodeCommandResponse(payload)
}

// Acknowledge sends an Acknowledge message to the MDM server.
// The cmdUUID is the UUID of the command to reference.
//
// The server can signal back with either a command to run
// or an empty (nil, nil) response body to end the communication
// (i.e. no commands to run).
func (c *TestAppleMDMClient) Acknowledge(cmdUUID string) (*mdm.Command, error) {
	payload := map[string]any{
		"Status":       "Acknowledged",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"CommandUUID":  cmdUUID,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	return c.sendAndDecodeCommandResponse(payload)
}

// NotNow sends a NotNow message to the MDM server.
// The cmdUUID is the UUID of the command to reference.
//
// The server can signal back with either a command to run
// or an empty (nil, nil) response body to end the communication
// (i.e. no commands to run).
func (c *TestAppleMDMClient) NotNow(cmdUUID string) (*mdm.Command, error) {
	payload := map[string]any{
		"Status":       "NotNow",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"CommandUUID":  cmdUUID,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) AcknowledgeDeviceInformation(udid, cmdUUID, deviceName, productName, timeZone string) (*mdm.Command, error) {
	payload := map[string]any{
		"Status":      "Acknowledged",
		"UDID":        udid,
		"CommandUUID": cmdUUID,
		"QueryResponses": map[string]interface{}{
			"AvailableDeviceCapacity": float64(51.53312768),
			"DeviceCapacity":          float64(64),
			"DeviceName":              deviceName,
			"OSVersion":               "17.5.1",
			"ProductName":             productName,
			"WiFiMAC":                 "ff:ff:ff:ff:ff:ff",
			"IsMDMLostModeEnabled":    false,
			"TimeZone":                timeZone,
		},
	}
	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) AcknowledgeDeviceLocation(udid, cmdUUID string, lat, long float64) (*mdm.Command, error) {
	payload := map[string]any{
		"Status":      "Acknowledged",
		"UDID":        udid,
		"CommandUUID": cmdUUID,
		"Latitude":    lat,
		"Longitude":   long,
	}

	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) AcknowledgeInstalledApplicationList(udid, cmdUUID string, software []fleet.Software) (*mdm.Command, error) {
	mdmSoftware := make([]map[string]interface{}, 0, len(software))
	for _, s := range software {
		mdmSoftware = append(mdmSoftware, map[string]interface{}{
			"Name":         s.Name,
			"ShortVersion": s.Version,
			"Identifier":   s.BundleIdentifier,
			"Installing":   !s.Installed,
		})
	}

	payload := map[string]any{
		"Status":                   "Acknowledged",
		"UDID":                     udid,
		"CommandUUID":              cmdUUID,
		"InstalledApplicationList": mdmSoftware,
	}

	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) AcknowledgeCertificateList(udid, cmdUUID string, certTemplates []*x509.Certificate) (*mdm.Command, error) {
	var certList []fleet.MDMAppleCertificateListItem
	for _, cert := range certTemplates {
		b, _, err := mysql.GenerateTestCertBytes(cert)
		if err != nil {
			return nil, err
		}
		certList = append(certList, fleet.MDMAppleCertificateListItem{
			CommonName: cert.Subject.CommonName,
			Data:       b,
		})
	}
	cmd := map[string]any{
		"CommandUUID":     cmdUUID,
		"UDID":            udid,
		"Status":          "Acknowledged",
		"CertificateList": certList,
	}

	return c.sendAndDecodeCommandResponse(cmd)
}

func (c *TestAppleMDMClient) GetBootstrapToken() ([]byte, error) {
	payload := map[string]any{
		"MessageType":  "GetBootstrapToken",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	res, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	if res.ContentLength == 0 {
		if c.debug {
			fmt.Printf("response: no bootstrap token returned\n")
		}
		return nil, nil
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if c.debug {
		fmt.Printf("response: %s\n", raw)
	}
	if err = res.Body.Close(); err != nil {
		return nil, fmt.Errorf("close response body: %w", err)
	}
	var p mdm.BootstrapToken
	err = plist.Unmarshal(raw, &p)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bootstrap token payload: %w", err)
	}

	return p.BootstrapToken, nil
}

// Err sends an Error message to the MDM server.
// The cmdUUID is the UUID of the command to reference.
//
// The server can signal back with either a command to run
// or an empty (nil, nil) response body to end the communication
// (i.e. no commands to run).
func (c *TestAppleMDMClient) Err(cmdUUID string, errChain []mdm.ErrorChain) (*mdm.Command, error) {
	payload := map[string]any{
		"Status":       "Error",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"CommandUUID":  cmdUUID,
		"ErrorChain":   errChain,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	return c.sendAndDecodeCommandResponse(payload)
}

// UserChannelErr sends an Error message to the MDM server on the user
// channel. UserEnroll must have been called first so that c.UserUUID is set.
func (c *TestAppleMDMClient) UserChannelErr(cmdUUID string, errChain []mdm.ErrorChain) (*mdm.Command, error) {
	if c.UserUUID == "" {
		return nil, errors.New("user UUID must be set for user channel error response")
	}
	payload := map[string]any{
		"Status":       "Error",
		"Topic":        "com.apple.mgmt.External." + c.Identifier(),
		"EnrollmentID": "testenrollmentid-" + c.Identifier(),
		"CommandUUID":  cmdUUID,
		"ErrorChain":   errChain,
		"UserID":       c.UserUUID,
	}
	if c.UUID != "" {
		payload["UDID"] = c.UUID
	}
	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) sendAndDecodeCommandResponse(payload map[string]any) (*mdm.Command, error) {
	res, err := c.request("", payload)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	if res.ContentLength == 0 {
		if c.debug {
			fmt.Printf("response: no commands returned\n")
		}
		return nil, nil
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if c.debug {
		fmt.Printf("response: %s\n", raw)
	}
	if err = res.Body.Close(); err != nil {
		return nil, fmt.Errorf("close response body: %w", err)
	}
	cmd, err := mdm.DecodeCommand(raw)
	if err != nil {
		return nil, fmt.Errorf("decode command: %w", err)
	}
	var p mdm.Command
	err = plist.Unmarshal(cmd.Raw, &p)
	if err != nil {
		return nil, fmt.Errorf("unmarshal command payload: %w", err)
	}
	p.Raw = cmd.Raw
	return &p, nil
}

func (c *TestAppleMDMClient) getSignerCertAndKey() (*x509.Certificate, crypto.PrivateKey, error) {
	if c.scepCert != nil && c.scepKey != nil {
		return c.scepCert, c.scepKey, nil
	}
	if c.acmeCert != nil && c.acmeKey != nil {
		return c.acmeCert, c.acmeKey, nil
	}
	return nil, nil, errors.New("no signer certificate and key available")
}

func (c *TestAppleMDMClient) request(contentType string, payload map[string]any) (*http.Response, error) {
	body, err := plist.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	signedData, err := pkcs7.NewSignedData(body)
	if err != nil {
		return nil, fmt.Errorf("create signed data: %w", err)
	}
	cert, key, err := c.getSignerCertAndKey()
	if err != nil {
		return nil, fmt.Errorf("get signer certificate and key: %w", err)
	}
	err = signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{})
	if err != nil {
		return nil, fmt.Errorf("add signer: %w", err)
	}
	sig, err := signedData.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish signing: %w", err)
	}

	if c.debug {
		fmt.Printf("request: %s\n", body)
	}
	request, err := http.NewRequest("POST", c.EnrollInfo.MDMURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Mdm-Signature", base64.StdEncoding.EncodeToString(sig))

	if c.fetchEnrollmentProfileFromMDMBYOD && c.authorizationBearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.authorizationBearerToken)
	}
	// #nosec (this client is used for testing only)
	cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}))
	response, err := cc.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}
	return response, nil
}

// ParseEnrollmentProfile parses the enrollment profile and returns the parsed information as EnrollInfo.
func ParseEnrollmentProfile(mobileConfig []byte) (*AppleEnrollInfo, error) {
	var enrollmentProfile struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	if err := plist.Unmarshal(mobileConfig, &enrollmentProfile); err != nil {
		return nil, fmt.Errorf("unmarshal enrollment profile: %w", err)
	}

	mdmURL, ok := enrollmentProfile.PayloadContent[1]["ServerURL"].(string)
	if !ok || mdmURL == "" {
		return nil, errors.New("MDM ServerURL field not found")
	}
	// Check the server sent a proper APNS topic.
	if apnsTopic, ok := enrollmentProfile.PayloadContent[1]["Topic"].(string); !ok || apnsTopic == "" {
		return nil, errors.New("MDM Topic field not found")
	}

	// assignedManagedAppleID is optional and only present in account driven enrollment flows so
	// only use it if it exists.
	var assignedManagedAppleID string
	assignedManagedAppleIDVal, ok := enrollmentProfile.PayloadContent[1]["AssignedManagedAppleID"]
	if ok {
		assignedManagedAppleID = assignedManagedAppleIDVal.(string)
	}

	enrollInfo := &AppleEnrollInfo{
		MDMURL:                 mdmURL,
		AssignedManagedAppleID: assignedManagedAppleID,
	}

	var err error
	payloadContent, ok := enrollmentProfile.PayloadContent[0]["PayloadContent"].(map[string]any)
	if ok {
		enrollInfo, err = parseSCEPEnrollmentPayload(*enrollInfo, payloadContent)
		if err != nil {
			return nil, err
		}
	} else {
		// Check for ACME
		_, ok := enrollmentProfile.PayloadContent[0]["DirectoryURL"].(string)
		_, ok2 := enrollmentProfile.PayloadContent[0]["HardwareBound"].(bool)
		_, ok3 := enrollmentProfile.PayloadContent[0]["Attest"].(bool)
		if !ok || !ok2 || !ok3 {
			// One of ACME fields are not present (we don't care about the value, but they need to be present)
			return nil, errors.New("not a valid ACME or SCEP enrollment profile")
		}

		enrollInfo, err = parseACMEEnrollmentPayload(*enrollInfo, enrollmentProfile.PayloadContent[0])
		if err != nil {
			return nil, err
		}
	}

	return enrollInfo, nil
}

func parseSCEPEnrollmentPayload(enrollInfo AppleEnrollInfo, payloadContent map[string]any) (*AppleEnrollInfo, error) {
	scepChallenge, ok := payloadContent["Challenge"].(string)
	if !ok || scepChallenge == "" {
		return nil, errors.New("SCEP Challenge field not found")
	}
	scepURL, ok := payloadContent["URL"].(string)
	if !ok || scepURL == "" {
		return nil, errors.New("SCEP URL field not found")
	}

	enrollInfo.SCEPChallenge = scepChallenge
	enrollInfo.SCEPURL = scepURL
	return &enrollInfo, nil
}

func parseACMEEnrollmentPayload(enrollInfo AppleEnrollInfo, payloadContent map[string]any) (*AppleEnrollInfo, error) {
	directoryURL, ok := payloadContent["DirectoryURL"].(string)
	if !ok || directoryURL == "" {
		return nil, errors.New("ACME DirectoryURL field not found")
	}

	// TODO CLEAN UP
	/*
			PayloadIdentifier: BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1
		  PayloadUUID: BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1
		  ClientIdentifier: 1c0a81b9-1f30-4393-aa5d-0d4064640233
		  DirectoryURL: http://127.0.0.1:63768/api/mdm/acme/0639af7d-7009-4ac2-9b73-839984323b68/directory
		  PayloadDisplayName: Fleet Identity ACME
		  PayloadType: com.apple.security.acme
		  HardwareBound: true
		  KeySize: 384
		  PayloadVersion: 1
		  Subject: [[[CN %SerialNumber%]]]
		  KeyType: ECSECPrimeRandom
		  Attest: true
	*/

	// TODO: Directory URL or just base URL with identifier
	enrollInfo.ACMEURL = directoryURL
	return &enrollInfo, nil
}

// numbers plus capital letters without I, L, O for readability
const serialLetters = "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"

// RandSerialNumber returns a fake random serial number.
func RandSerialNumber() string {
	return randStr(12)
}

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		//nolint:gosec // not used for crypto, only to generate random serial for testing
		b[i] = serialLetters[mrand.Intn(len(serialLetters))]
	}
	return string(b)
}

// RandUDID returns a fake random iOS/iPadOS 17+ UDID.
func RandUDID() string {
	return fmt.Sprintf("%s-%s", randStr(8), randStr(16))
}

type scepClient interface {
	scepserver.Service
	Supports(capacity string) bool
}

func newSCEPClient(
	serverURL string,
	logger *slog.Logger,
) (scepClient, error) {
	endpoints, err := makeClientSCEPEndpoints(serverURL)
	if err != nil {
		return nil, err
	}
	endpoints.GetEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.GetEndpoint)
	endpoints.PostEndpoint = scepserver.EndpointLoggingMiddleware(logger)(endpoints.PostEndpoint)
	return endpoints, nil
}

// makeClientSCEPClientEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the remote instance, via a transport/http.Client.
func makeClientSCEPEndpoints(instance string) (*scepserver.Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	// #nosec (this client is used for testing only)
	c := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}))
	options := []httptransport.ClientOption{
		httptransport.SetClient(c),
	}

	return &scepserver.Endpoints{
		GetEndpoint: httptransport.NewClient(
			"GET",
			tgt,
			scepserver.EncodeSCEPRequest,
			scepserver.DecodeSCEPResponse,
			options...).Endpoint(),
		PostEndpoint: httptransport.NewClient(
			"POST",
			tgt,
			scepserver.EncodeSCEPRequest,
			scepserver.DecodeSCEPResponse,
			options...).Endpoint(),
	}, nil
}

// EncodeDeviceInfo is a helper function to provide mock device info for the x-aspen-deviceinfo
// header that is sent by the device during the Apple MDM enrollment process.
func EncodeDeviceInfo(machineInfo fleet.MDMAppleMachineInfo) (string, error) {
	sig, err := MachineInfoAsPKCS7(machineInfo)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(sig), nil
}

func AccountDrivenUserEnrollDeviceInfoAsPKCS7(deviceInfo fleet.MDMAppleAccountDrivenUserEnrollDeviceInfo) ([]byte, error) {
	return appleInfoStructAsPKCS7(deviceInfo)
}

// MachineInfoAsPKCS7 marshals and signs Apple's machine info.
func MachineInfoAsPKCS7(machineInfo fleet.MDMAppleMachineInfo) ([]byte, error) {
	return appleInfoStructAsPKCS7(machineInfo)
}

func appleInfoStructAsPKCS7(v interface{}) ([]byte, error) {
	body, err := plist.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal device info: %w", err)
	}

	// body is expected to be a PKCS7 signed message, although we don't currently verify the signature
	signedData, err := pkcs7.NewSignedData(body)
	if err != nil {
		return nil, fmt.Errorf("create signed data: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA private key: %w", err)
	}
	crtBytes, err := depot.NewCACert().SelfSign(rand.Reader, key.Public(), key)
	if err != nil {
		return nil, fmt.Errorf("create self-signed certificate: %w", err)
	}
	crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, fmt.Errorf("parse self-signed certificate: %w", err)
	}
	if err := signedData.AddSigner(crt, key, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("add signer: %w", err)
	}
	sig, err := signedData.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish signing: %w", err)
	}
	return sig, nil
}
