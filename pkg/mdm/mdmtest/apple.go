package mdmtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/cryptoutil/x509util"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/google/uuid"
	"github.com/groob/plist"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"go.mozilla.org/pkcs7"
)

// TestAppleMDMClient simulates a macOS MDM client.
type TestAppleMDMClient struct {
	// UUID is a random fake unique ID of the simulated device.
	UUID string
	// SerialNumber is a random fake serial number of the simulated device.
	SerialNumber string
	// Model is the model of the simulated device.
	Model string

	// EnrollInfo holds the information necessary to enroll to an MDM server.
	EnrollInfo AppleEnrollInfo

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
	// desktopURLToken is the token used to fetch the enrollment profile
	// from Fleet as if it were a device running the DEP flow.
	depURLToken string

	// scepCert contains the SCEP client certificate generated during the
	// SCEP enrollment process.
	scepCert *x509.Certificate
	// scepKey contains the SCEP client private key generated during the
	// SCEP enrollment process.
	scepKey *rsa.PrivateKey
}

// TestMDMAppleClientOption allows configuring a TestMDMClient.
type TestMDMAppleClientOption func(*TestAppleMDMClient)

// TestMDMAppleClientDebug configures the TestMDMClient to run in debug mode.
func TestMDMAppleClientDebug() TestMDMAppleClientOption {
	return func(c *TestAppleMDMClient) {
		c.debug = true
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

// NewTestMDMClientDEP will create a simulated device that will not fetch the enrollment
// profile from Fleet. The enrollment information is to be provided in the enrollInfo.
func NewTestMDMClientAppleDirect(enrollInfo AppleEnrollInfo, opts ...TestMDMAppleClientOption) *TestAppleMDMClient {
	c := TestAppleMDMClient{
		UUID:         strings.ToUpper(uuid.New().String()),
		SerialNumber: RandSerialNumber(),
		Model:        "MacBookPro16,1",

		EnrollInfo: enrollInfo,
	}
	for _, fn := range opts {
		fn(&c)
	}
	return &c
}

// Enroll runs the MDM enroll protocol on the simulated device.
func (c *TestAppleMDMClient) Enroll() error {
	switch {
	case c.fetchEnrollmentProfileFromDesktop:
		if err := c.fetchEnrollmentProfileFromDesktopURL(); err != nil {
			return fmt.Errorf("get enrollment profile from desktop URL: %w", err)
		}
	case c.fetchEnrollmentProfileFromDEP:
		if err := c.fetchEnrollmentProfileFromDEPURL(); err != nil {
			return fmt.Errorf("get enrollment profile from DEP URL: %w", err)
		}
	default:
		if c.EnrollInfo.SCEPURL == "" || c.EnrollInfo.MDMURL == "" || c.EnrollInfo.SCEPChallenge == "" {
			return fmt.Errorf("missing info needed to perform enrollment: %+v", c.EnrollInfo)
		}
	}
	if err := c.SCEPEnroll(); err != nil {
		return fmt.Errorf("scep enroll: %w", err)
	}
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	if err := c.TokenUpdate(); err != nil {
		return fmt.Errorf("token update: %w", err)
	}
	return nil
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromDesktopURL() error {
	return c.fetchEnrollmentProfile(
		"/api/latest/fleet/device/" + c.desktopURLToken + "/mdm/apple/manual_enrollment_profile",
	)
}

func (c *TestAppleMDMClient) fetchEnrollmentProfileFromDEPURL() error {
	return c.fetchEnrollmentProfile(
		apple_mdm.EnrollPath + "?token=" + c.depURLToken,
	)
}

func (c *TestAppleMDMClient) fetchEnrollmentProfile(path string) error {
	request, err := http.NewRequest("GET", c.fleetServerURL+path, nil)
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
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if err := response.Body.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	enrollInfo, err := ParseEnrollmentProfile(body)
	if err != nil {
		return fmt.Errorf("parse enrollment profile: %w", err)
	}
	c.EnrollInfo = *enrollInfo

	return nil
}

// SCEPEnroll runs the SCEP enroll protocol for the simulated device.
func (c *TestAppleMDMClient) SCEPEnroll() error {
	ctx := context.Background()

	var logger log.Logger
	if c.debug {
		logger = kitlog.NewJSONLogger(os.Stdout)
	} else {
		logger = kitlog.NewNopLogger()
	}
	client, err := newSCEPClient(c.EnrollInfo.SCEPURL, logger)
	if err != nil {
		return fmt.Errorf("scep client: %w", err)
	}

	// (1). Get the CA certificate from the SCEP server.
	resp, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return fmt.Errorf("get CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificates(resp)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	// (2). Generate RSA key pair.
	devicePrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate RSA private key: %w", err)
	}

	// (3). Generate CSR.
	cn := fmt.Sprintf("fleet-testdevice-%s", c.UUID)
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   cn,
				Organization: []string{"fleet-organization"},
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: c.EnrollInfo.SCEPChallenge,
	}
	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, devicePrivateKey)
	if err != nil {
		return fmt.Errorf("create CSR: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return fmt.Errorf("parse CSR: %w", err)
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
		return fmt.Errorf("generate cert serial number: %w", err)
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
		return fmt.Errorf("create device certificate: %w", err)
	}
	deviceCertificateForRequest, err := x509.ParseCertificate(deviceCertificateDerBytes)
	if err != nil {
		return fmt.Errorf("parse device certificate: %w", err)
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
	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("create CSR request: %w", err)
	}
	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return fmt.Errorf("do CSR request: %w", err)
	}
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(logger), scep.WithCACerts(msg.Recipients))
	if err != nil {
		return fmt.Errorf("parse PKIMessage response: %w", err)
	}
	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		return fmt.Errorf("PKIMessage CSR request failed with code: %s, fail info: %s", pkiMsgResp.PKIStatus, pkiMsgResp.FailInfo)
	}
	if err := pkiMsgResp.DecryptPKIEnvelope(deviceCertificateForRequest, devicePrivateKey); err != nil {
		return fmt.Errorf("decrypt PKI envelope: %w", err)
	}

	// (6). Finally, set the signed certificate returned from the server as the device certificate and key.
	c.scepCert = pkiMsgResp.CertRepMessage.Certificate
	c.scepKey = devicePrivateKey

	if c.debug {
		fmt.Println("SCEP enrollment successful")
	}

	return nil
}

// Authenticate sends the Authenticate message to the MDM server (Check In protocol).
func (c *TestAppleMDMClient) Authenticate() error {
	payload := map[string]any{
		"MessageType":  "Authenticate",
		"UDID":         c.UUID,
		"Model":        c.Model,
		"DeviceName":   "testdevice" + c.SerialNumber,
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
		"SerialNumber": c.SerialNumber,
	}
	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// TokenUpdate sends the TokenUpdate message to the MDM server (Check In protocol).
func (c *TestAppleMDMClient) TokenUpdate() error {
	payload := map[string]any{
		"MessageType":  "TokenUpdate",
		"UDID":         c.UUID,
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
		"NotOnConsole": "false",
		"PushMagic":    "pushmagic" + c.SerialNumber,
		"Token":        []byte("token" + c.SerialNumber),
	}
	_, err := c.request("application/x-apple-aspen-mdm-checkin", payload)
	return err
}

// Checkout sends the CheckOut message to the MDM server.
func (c *TestAppleMDMClient) Checkout() error {
	payload := map[string]any{
		"MessageType":  "CheckOut",
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"UDID":         c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
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
func (c *TestAppleMDMClient) Idle() (*micromdm.CommandPayload, error) {
	payload := map[string]any{
		"Status":       "Idle",
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"UDID":         c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
	}
	return c.sendAndDecodeCommandResponse(payload)
}

// Acknowledge sends an Acknowledge message to the MDM server.
// The cmdUUID is the UUID of the command to reference.
//
// The server can signal back with either a command to run
// or an empty (nil, nil) response body to end the communication
// (i.e. no commands to run).
func (c *TestAppleMDMClient) Acknowledge(cmdUUID string) (*micromdm.CommandPayload, error) {
	payload := map[string]any{
		"Status":       "Acknowledged",
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"UDID":         c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
		"CommandUUID":  cmdUUID,
	}
	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) GetBootstrapToken() ([]byte, error) {
	payload := map[string]any{
		"MessageType":  "GetBootstrapToken",
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"UDID":         c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
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
func (c *TestAppleMDMClient) Err(cmdUUID string, errChain []mdm.ErrorChain) (*micromdm.CommandPayload, error) {
	payload := map[string]any{
		"Status":       "Error",
		"Topic":        "com.apple.mgmt.External." + c.UUID,
		"UDID":         c.UUID,
		"EnrollmentID": "testenrollmentid-" + c.UUID,
		"CommandUUID":  cmdUUID,
		"ErrorChain":   errChain,
	}
	return c.sendAndDecodeCommandResponse(payload)
}

func (c *TestAppleMDMClient) sendAndDecodeCommandResponse(payload map[string]any) (*micromdm.CommandPayload, error) {
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
	var p micromdm.CommandPayload
	err = plist.Unmarshal(cmd.Raw, &p)
	if err != nil {
		return nil, fmt.Errorf("unmarshal command payload: %w", err)
	}
	return &p, nil
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
	err = signedData.AddSigner(c.scepCert, c.scepKey, pkcs7.SignerInfoConfig{})
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
		PayloadContent []map[string]interface{} `plist:"PayloadContent"`
	}
	if err := plist.Unmarshal(mobileConfig, &enrollmentProfile); err != nil {
		return nil, fmt.Errorf("unmarshal enrollment profile: %w", err)
	}
	payloadContent := enrollmentProfile.PayloadContent[0]["PayloadContent"].(map[string]interface{})

	scepChallenge, ok := payloadContent["Challenge"].(string)
	if !ok || scepChallenge == "" {
		return nil, errors.New("SCEP Challenge field not found")
	}
	scepURL, ok := payloadContent["URL"].(string)
	if !ok || scepURL == "" {
		return nil, errors.New("SCEP URL field not found")
	}
	mdmURL, ok := enrollmentProfile.PayloadContent[1]["ServerURL"].(string)
	if !ok || mdmURL == "" {
		return nil, errors.New("MDM ServerURL field not found")
	}
	// Check the server sent a proper APNS topic.
	if apnsTopic, ok := enrollmentProfile.PayloadContent[1]["Topic"].(string); !ok || apnsTopic == "" {
		return nil, errors.New("MDM Topic field not found")
	}
	return &AppleEnrollInfo{
		SCEPChallenge: scepChallenge,
		SCEPURL:       scepURL,
		MDMURL:        mdmURL,
	}, nil
}

// numbers plus capital letters without I, L, O for readability
const serialLetters = "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"

// RandSerialNumber returns a fake random serial number.
func RandSerialNumber() string {
	b := make([]byte, 12)
	for i := range b {
		//nolint:gosec // not used for crypto, only to generate random serial for testing
		b[i] = serialLetters[mrand.Intn(len(serialLetters))]
	}
	return string(b)
}

type scepClient interface {
	scepserver.Service
	Supports(cap string) bool
}

func newSCEPClient(
	serverURL string,
	logger log.Logger,
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
