package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	_              scepserver.ServiceWithIdentifier = (*scepProxyService)(nil)
	challengeRegex                                  = regexp.MustCompile(`(?i)The enrollment challenge password is: <B> (?P<password>\S*)`)
)

const (
	fullPasswordCache              = "The password cache is full."
	ndesInsufficientPermissions    = "You do not have sufficient permission to enroll with SCEP."
	MessageSCEPProxyNotConfigured  = "SCEP proxy is not configured"
	NDESChallengeInvalidAfter      = 57 * time.Minute
	SmallstepChallengeInvalidAfter = 4 * time.Minute
)

// decodeHTMLResponse decodes HTTP response body to a string, handling various encodings.
// Windows NDES servers return UTF-16 LE (often without BOM), while Okta returns UTF-8.
//
// Detection order:
//  1. UTF-16 LE heuristic (for Windows NDES compatibility; checked first because
//     charset detection libraries can't parse meta tags in UTF-16 encoded content)
//  2. Content-Type charset header and BOM detection via charset.DetermineEncoding
//  3. Fall back to UTF-8
func decodeHTMLResponse(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	// Check for UTF-16 LE first. Windows NDES servers return UTF-16 LE without BOM
	// and without proper Content-Type charset. We must detect this before trying
	// charset.DetermineEncoding, which would fail to parse meta tags in UTF-16 bytes.
	if looksLikeUTF16LE(body) {
		// Use BOMOverride to handle BOM if present (strips it from output)
		utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
		decoder := unicode.BOMOverride(utf16le.NewDecoder())
		if decoded, _, err := transform.Bytes(decoder, body); err == nil {
			return string(decoded)
		}
	}

	// Standard HTML5 encoding detection:
	// - Checks Content-Type charset parameter
	// - Detects BOM (Byte Order Mark)
	// - Prescans for meta charset tags
	enc, _, _ := charset.DetermineEncoding(body, contentType)
	if enc != nil {
		if decoded, _, err := transform.Bytes(enc.NewDecoder(), body); err == nil {
			return string(decoded)
		}
	}

	// Default: treat as UTF-8
	return string(body)
}

// looksLikeUTF16LE checks if the body appears to be UTF-16 LE encoded.
// Detection is done in order of reliability:
// 1. BOM (Byte Order Mark) - most authoritative
// 2. HTML pattern - UTF-16 LE HTML starts with '<' 0x00
// 3. Null byte percentage - fallback heuristic
func looksLikeUTF16LE(body []byte) bool {
	if len(body) < 4 {
		return false
	}

	// 1. Check for UTF-16 LE BOM (FF FE) - most reliable indicator
	if body[0] == 0xFF && body[1] == 0xFE {
		return true
	}

	// 2. Check for HTML pattern: '<' followed by 0x00
	// HTML content starts with '<' (e.g., "<HTML>", "<!DOCTYPE>")
	// In UTF-16 LE, this becomes 0x3C 0x00 - very unlikely in valid UTF-8
	if body[0] == '<' && body[1] == 0x00 {
		return true
	}

	// 3. Fallback: count null bytes at odd positions
	// UTF-16 LE ASCII has null at every odd position (char, 0x00, char, 0x00, ...)
	// Require 90% to reduce false positives from UTF-8 with occasional nulls

	// utf16DetectionSampleSize is the number of bytes to examine when detecting UTF-16 LE encoding.
	const utf16DetectionSampleSize = 100
	checkLen := min(len(body), utf16DetectionSampleSize)
	nullCount := 0
	for i := 1; i < checkLen; i += 2 {
		if body[i] == 0x00 {
			nullCount++
		}
	}
	checked := checkLen / 2
	return checked > 0 && float64(nullCount)/float64(checked) >= 0.9 // 90%
}

// scepCertificateRequest abstracts the common operations needed for SCEP certificate
// requests across different platforms (Apple, Windows, Android).
type scepCertificateRequest interface {
	// GetStatus returns the delivery status of the certificate request.
	// Returns empty string if status is not set.
	GetStatus() fleet.MDMDeliveryStatus
	// GetChallengeRetrievedAt returns when the challenge was retrieved (for expiration checks).
	GetChallengeRetrievedAt() *time.Time
	// GetCAType returns the certificate authority type (NDES, Smallstep, CustomSCEPProxy).
	GetCAType() fleet.CAConfigAssetType
	// GetCAName returns the name of the certificate authority.
	GetCAName() string
	// GetProfileUUID returns the profile/template UUID.
	GetProfileUUID() string
}

// hostMDMCertificateProfileAdapter adapts HostMDMCertificateProfile to scepCertificateRequest.
type hostMDMCertificateProfileAdapter struct {
	profile *fleet.HostMDMCertificateProfile
}

func (a *hostMDMCertificateProfileAdapter) GetStatus() fleet.MDMDeliveryStatus {
	if a.profile.Status == nil {
		return ""
	}
	return *a.profile.Status
}

func (a *hostMDMCertificateProfileAdapter) GetChallengeRetrievedAt() *time.Time {
	return a.profile.ChallengeRetrievedAt
}

func (a *hostMDMCertificateProfileAdapter) GetCAType() fleet.CAConfigAssetType {
	return a.profile.Type
}

func (a *hostMDMCertificateProfileAdapter) GetCAName() string {
	return a.profile.CAName
}

func (a *hostMDMCertificateProfileAdapter) GetProfileUUID() string {
	return a.profile.ProfileUUID
}

// certificateTemplateForHostAdapter adapts CertificateTemplateForHost to scepCertificateRequest.
type certificateTemplateForHostAdapter struct {
	template    *fleet.CertificateTemplateForHost
	profileUUID string
}

func (a *certificateTemplateForHostAdapter) GetStatus() fleet.MDMDeliveryStatus {
	if a.template.Status == nil {
		return ""
	}
	return fleet.CertificateTemplateStatusToMDMDeliveryStatus(*a.template.Status)
}

func (a *certificateTemplateForHostAdapter) GetChallengeRetrievedAt() *time.Time {
	// Android certificate templates don't track challenge retrieval time;
	// they use one-time fleet challenges validated via ConsumeChallenge.
	return nil
}

func (a *certificateTemplateForHostAdapter) GetCAType() fleet.CAConfigAssetType {
	return a.template.CAType
}

func (a *certificateTemplateForHostAdapter) GetCAName() string {
	return a.template.CAName
}

func (a *certificateTemplateForHostAdapter) GetProfileUUID() string {
	return a.profileUUID
}

type scepProxyService struct {
	ds fleet.Datastore
	// info logging is implemented in the service middleware layer.
	debugLogger log.Logger
	Timeout     *time.Duration
}

// NewSCEPProxyService creates a new scep proxy service
func NewSCEPProxyService(ds fleet.Datastore, logger log.Logger, timeout *time.Duration) scepserver.ServiceWithIdentifier {
	if timeout == nil {
		timeout = ptr.Duration(30 * time.Second)
	}
	return &scepProxyService{
		ds:          ds,
		debugLogger: logger,
		Timeout:     timeout,
	}
}

// GetCACaps returns a list of SCEP options which are supported by the server.
// It is a pass-through call to the SCEP server.
func (svc *scepProxyService) GetCACaps(ctx context.Context, identifier string) ([]byte, error) {
	scepURL, err := svc.validateIdentifier(ctx, identifier, false)
	if err != nil {
		return nil, err
	}

	client, err := scepclient.New(scepURL, svc.debugLogger, scepclient.WithTimeout(svc.Timeout))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, err := client.GetCACaps(ctx)
	if err != nil {
		return res, ctxerr.Wrapf(ctx, err, "Could not GetCACaps from SCEP server %s", scepURL)
	}
	return res, nil
}

// GetCACert returns the CA certificate(s) from SCEP server.
// It is a pass-through call to the SCEP server.
func (svc *scepProxyService) GetCACert(ctx context.Context, message string, identifier string) ([]byte, int, error) {
	scepURL, err := svc.validateIdentifier(ctx, identifier, false)
	if err != nil {
		return nil, 0, err
	}

	client, err := scepclient.New(scepURL, svc.debugLogger, scepclient.WithTimeout(svc.Timeout))
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, num, err := client.GetCACert(ctx, message)
	if err != nil {
		return res, num, ctxerr.Wrapf(ctx, err, "Could not GetCACert from SCEP server %s", scepURL)
	}
	return res, num, nil
}

// NOTE: Any changes to this method must ensure that the challenge portion of the identifer is
// properly validated using the before proceeding with the PKIOperation.
func (svc *scepProxyService) PKIOperation(ctx context.Context, data []byte, identifier string) ([]byte, error) {
	// We only check for expired NDES challenge during this (the last) SCEP request to account for previous requests having large network delays
	scepURL, err := svc.validateIdentifier(ctx, identifier, true) // checkChallenge must be true to validate the challenge portion of the identifier
	if err != nil {
		return nil, err
	}

	client, err := scepclient.New(scepURL, svc.debugLogger, scepclient.WithTimeout(svc.Timeout))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating SCEP client")
	}
	res, err := client.PKIOperation(ctx, data)
	if err != nil {
		return res, ctxerr.Wrapf(ctx, err,
			"Could not do PKIOperation on SCEP server %s", scepURL)
	}
	return res, nil
}

func (svc *scepProxyService) validateIdentifier(ctx context.Context, identifier string, checkChallenge bool) (string,
	error,
) {
	groupedCAs, err := svc.ds.GetGroupedCertificateAuthorities(ctx, false)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	parsedID, err := url.PathUnescape(identifier)
	if err != nil {
		// Should never happen since the identifier comes in as a path variable
		return "", ctxerr.Wrap(ctx, err, "unescaping identifier in URL path")
	}
	parsedIDs := strings.Split(parsedID, ",")
	if len(parsedIDs) < 2 || parsedIDs[0] == "" || parsedIDs[1] == "" {
		// Return error that implements kithttp.StatusCoder interface
		return "", &scepserver.BadRequestError{Message: "invalid identifier in URL path"}
	}
	hostUUID := parsedIDs[0]
	profileUUID := parsedIDs[1]
	caName := "NDES" // default
	if len(parsedIDs) > 2 {
		caName = parsedIDs[2]
	}
	var fleetChallenge string
	if len(parsedIDs) > 3 {
		fleetChallenge = parsedIDs[3]
	}

	if !strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix) &&
		!strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix) &&
		!strings.HasPrefix(profileUUID, fleet.MDMAndroidProfileUUIDPrefix) {
		return "", &scepserver.BadRequestError{Message: fmt.Sprintf("invalid profile UUID (only Apple, Windows, and Android config profiles are supported): %s",
			profileUUID)}
	}

	var certReq scepCertificateRequest

	switch {
	case strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix):
		profile, err := svc.ds.GetAppleHostMDMCertificateProfile(ctx, hostUUID, profileUUID, caName)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting host MDM profile")
		}
		if profile != nil {
			certReq = &hostMDMCertificateProfileAdapter{profile: profile}
		}

	case strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix):
		profile, err := svc.ds.GetWindowsHostMDMCertificateProfile(ctx, hostUUID, profileUUID, caName)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting host MDM profile")
		}
		if profile != nil {
			certReq = &hostMDMCertificateProfileAdapter{profile: profile}
		}

	case strings.HasPrefix(profileUUID, fleet.MDMAndroidProfileUUIDPrefix):
		// Android identifier format: {hostUUID},g{certificateTemplateID},{caType},{challenge}
		// Parse the certificate template ID from the profileUUID (e.g., "g123" -> 123)
		certTemplateIDStr := strings.TrimPrefix(profileUUID, fleet.MDMAndroidProfileUUIDPrefix)
		certTemplateID, err := strconv.ParseUint(certTemplateIDStr, 10, 32)
		if err != nil {
			return "", &scepserver.BadRequestError{Message: fmt.Sprintf("invalid Android certificate template ID: %s", certTemplateIDStr)}
		}

		template, err := svc.ds.GetCertificateTemplateForHost(ctx, hostUUID, uint(certTemplateID))
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting Android certificate template")
		}
		certReq = &certificateTemplateForHostAdapter{
			template:    template,
			profileUUID: profileUUID,
		}
		// Use the fleet challenge from the template if not provided in the identifier
		if fleetChallenge == "" && template.FleetChallenge != nil {
			fleetChallenge = *template.FleetChallenge
		}
	}

	if certReq == nil {
		// Return error that implements kithttp.StatusCoder interface
		return "", &scepserver.BadRequestError{Message: "unknown identifier in URL path"}
	}
	// We skip windows profiles for this check here as they instantly go to verifying when sent out, might change for windows renewal.
	if certReq.GetStatus() != fleet.MDMDeliveryPending && !strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix) {
		// This could happen if Fleet DB was updated before the profile was updated on the host.
		// We expect another certificate request from the host once the profile is updated.
		status := certReq.GetStatus()
		if status == "" {
			status = "null"
		}
		// FIXME: MDM client will report a failed status for the profile when we return bad request, which consumes the sole retry attempt.
		// Seems like we should proactively use ResendHostCertificateProfile here too?
		return "", &scepserver.BadRequestError{Message: fmt.Sprintf("profile status (%s) is not 'pending' for host:%s profile:%s", status,
			hostUUID, profileUUID)}
	}
	var scepURL string

	switch certReq.GetCAType() {
	case fleet.CAConfigNDES:
		if groupedCAs.NDESSCEP == nil {
			// Return error that implements kithttp.StatusCoder interface
			return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
		}
		challengeRetrievedAt := certReq.GetChallengeRetrievedAt()
		if checkChallenge && challengeRetrievedAt != nil && challengeRetrievedAt.Add(NDESChallengeInvalidAfter).Before(time.Now()) {
			// The challenge password was retrieved for this profile, and is now invalid.
			// We need to resend the profile with a new challenge password.
			// Note: we don't actually know if it is invalid, and we can't get that exact feedback from SCEP server.
			if err := svc.ds.ResendHostMDMProfile(ctx, hostUUID, profileUUID); err != nil {
				return "", ctxerr.Wrap(ctx, err, "resending host mdm profile")
			}
			return "", &scepserver.BadRequestError{Message: "challenge password has expired"}
		}
		scepURL = groupedCAs.NDESSCEP.URL

	case fleet.CAConfigSmallstep:
		if len(groupedCAs.Smallstep) < 1 {
			return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
		}
		for _, ca := range groupedCAs.Smallstep {
			if ca.Name == certReq.GetCAName() {
				scepURL = ca.URL
				break
			}
		}

		// FIXME: See comment in datastore method regarding how we resend profiles with dynamic content
		challengeRetrievedAt := certReq.GetChallengeRetrievedAt()
		if checkChallenge && challengeRetrievedAt != nil && challengeRetrievedAt.Add(SmallstepChallengeInvalidAfter).Before(time.Now()) {
			// The challenge password was retrieved for this profile, and is now invalid.
			// We need to resend the profile with a new challenge password.
			// Note: we don't actually know if it is invalid, and we can't get that exact feedback from SCEP server.
			if err := svc.ds.ResendHostCertificateProfile(ctx, hostUUID, profileUUID); err != nil {
				return "", ctxerr.Wrap(ctx, err, "resending host mdm profile")
			}
			return "", &scepserver.BadRequestError{Message: "challenge password has expired"}
		}

	case fleet.CAConfigCustomSCEPProxy:
		if len(groupedCAs.CustomScepProxy) < 1 {
			return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
		}
		for _, ca := range groupedCAs.CustomScepProxy {
			if ca.Name == certReq.GetCAName() {
				scepURL = ca.URL
				break
			}
		}

		if strings.HasPrefix(certReq.GetProfileUUID(), fleet.MDMWindowsProfileUUIDPrefix) {
			// TODO: Early return for Windows profiles as they do not support resending yet.
			return scepURL, nil
		}

		if checkChallenge {
			if err := svc.handleFleetChallenge(ctx, fleetChallenge, hostUUID, profileUUID); err != nil {
				// FIXME: The layered logging implementation of the scepProxyService not
				// intuitive. Can we make it so that we return fleet.ErrWithInternal to
				// better capture/log the context errors here?
				svc.debugLogger.Log(
					"msg", "custom scep proxy: failed to handle fleet challenge",
					"host_uuid", hostUUID,
					"profile_uuid", profileUUID,
					"err", err.Error(),
				)
				return "", &scepserver.BadRequestError{
					Message: "custom scep challenge failed",
				}
			}
		}
	}
	if scepURL == "" {
		return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
	}
	return scepURL, nil
}

func (svc *scepProxyService) GetNextCACert(_ context.Context) ([]byte, error) {
	// NDES on Windows Server 2022 does not support this, as advertised via GetCACaps
	return nil, errors.New("GetNextCACert is not implemented for SCEP proxy")
}

// handleFleetChallenge handles the validation of the fleet challenge for custom SCEP profiles as
// well as resending the profile if the challenge cannot be validated. If it is valid, it returns
// nil. If it cannot be validated or if any errors occur while validating or resending the profile,
// it returns a concatenated error.
//
// TODO: Consider refactoring to differentiate between invalid challenge and other errors. As it
// stands, we're resending the profile in both cases.
func (svc *scepProxyService) handleFleetChallenge(ctx context.Context, fleetChallenge string, hostUUID string, profileUUID string) error {
	isAndroid := strings.HasPrefix(profileUUID, fleet.MDMAndroidProfileUUIDPrefix)

	if err := svc.ds.ConsumeChallenge(ctx, fleetChallenge); err != nil {
		// For Android profiles, don't return an error when the challenge is not found.
		// This is likely a duplicate/retry request where the first request already succeeded.
		// Returning an error would cause the device to report "failed" status, incorrectly
		// overwriting the "verified" status from the successful first request.
		if isAndroid && errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		// For non-Android profiles, attempt to resend the profile
		// FIXME: See comment in datastore method regarding how we resend profiles with dynamic content
		var errs []error
		errs = append(errs, ctxerr.Wrap(ctx, err, "custom scep proxy: validating challenge"))
		if err := svc.ds.ResendHostCertificateProfile(ctx, hostUUID, profileUUID); err != nil {
			errs = append(errs, ctxerr.Wrap(ctx, err, "custom scep proxy: resending host mdm profile"))
		}
		return ctxerr.Wrap(ctx, errors.Join(errs...), "custom scep proxy: failed to handle fleet challenge")
	}

	return nil
}

type SCEPConfigService struct {
	logger log.Logger
	// Timeout is the timeout for SCEP requests.
	Timeout *time.Duration
}

func NewSCEPConfigService(logger log.Logger, timeout *time.Duration) fleet.SCEPConfigService {
	if timeout == nil {
		timeout = ptr.Duration(30 * time.Second)
	}
	return &SCEPConfigService{
		logger:  logger,
		Timeout: timeout,
	}
}

// Compile check that SCEPConfigService implements the interface.
var _ fleet.SCEPConfigService = (*SCEPConfigService)(nil)

func (s *SCEPConfigService) ValidateNDESSCEPAdminURL(ctx context.Context, proxy fleet.NDESSCEPProxyCA) error {
	_, err := s.GetNDESSCEPChallenge(ctx, proxy)
	return err
}

func (s *SCEPConfigService) GetNDESSCEPChallenge(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
	adminURL, username, password := proxy.AdminURL, proxy.Username, proxy.Password
	// Get the challenge from NDES
	client := fleethttp.NewClient(fleethttp.WithTimeout(*s.Timeout))
	client.Transport = ntlmssp.Negotiator{
		RoundTripper: fleethttp.NewTransport(),
	}
	req, err := http.NewRequest(http.MethodGet, adminURL, http.NoBody)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "creating request")
	}
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "sending request")
	}
	if resp.StatusCode != http.StatusOK {
		return "", ctxerr.Wrap(ctx, NDESInvalidError{msg: fmt.Sprintf(
			"unexpected status code: %d; could not retrieve the enrollment challenge password; invalid admin URL or credentials; please correct and try again",
			resp.StatusCode)})
	}
	defer resp.Body.Close()

	// Read raw bytes first to detect encoding
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "reading response body")
	}

	htmlString := decodeHTMLResponse(rawBody, resp.Header.Get("Content-Type"))

	matches := challengeRegex.FindStringSubmatch(htmlString)
	challenge := ""
	if matches != nil {
		challenge = matches[challengeRegex.SubexpIndex("password")]
	}
	if challenge == "" {
		switch {
		case strings.Contains(htmlString, fullPasswordCache):
			return "", ctxerr.Wrap(ctx,
				NewNDESPasswordCacheFullError("the password cache is full; please increase the number of cached passwords in NDES; by default, NDES caches 5 passwords and they expire 60 minutes after they are created"))
		case strings.Contains(htmlString, ndesInsufficientPermissions):
			return "", ctxerr.Wrap(ctx,
				NewNDESInsufficientPermissionsError("this account does not have sufficient permissions to enroll with SCEP. Please use a different account with NDES SCEP enroll permissions."))
		}
		return "", ctxerr.Wrap(ctx,
			NewNDESInvalidError("could not retrieve the enrollment challenge password; invalid admin URL or credentials; please correct and try again"))
	}
	return challenge, nil
}

func (s *SCEPConfigService) ValidateSCEPURL(ctx context.Context, url string) error {
	client, err := scepclient.New(url, s.logger, scepclient.WithTimeout(s.Timeout))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating SCEP client; invalid SCEP URL; please correct and try again")
	}

	certs, _, err := client.GetCACert(ctx, "")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "could not retrieve CA certificate from SCEP URL; invalid SCEP URL; please correct and try again")
	}
	if len(certs) == 0 {
		return ctxerr.New(ctx, "SCEP URL did not return a CA certificate")
	}
	return nil
}

func (s *SCEPConfigService) ValidateSmallstepChallengeURL(ctx context.Context, ca fleet.SmallstepSCEPProxyCA) error {
	_, err := s.GetSmallstepSCEPChallenge(ctx, ca)
	return err
}

func (s *SCEPConfigService) GetSmallstepSCEPChallenge(ctx context.Context, ca fleet.SmallstepSCEPProxyCA) (string, error) {
	// Get the challenge from Smallstep
	client := fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second))
	client.Transport = ntlmssp.Negotiator{
		RoundTripper: fleethttp.NewTransport(),
	}
	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(fleet.SmallstepChallengeRequestBody{
		Webhook: fleet.SmallstepChallengeWebhook{
			ID:             1,
			WebhookEvent:   "SCEPChallenge",
			EventTimestamp: time.Now().Unix(),
			Name:           "SCEPChallenge",
		},
		Event: fleet.SmallstepChallengeEvent{
			SCEPServerURL:     ca.URL,
			PayloadIdentifier: uuid.New().String(),
			PayloadTypes:      []string{"com.apple.security.scep"},
		},
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "encoding params as JSON")
	}
	req, err := http.NewRequest(http.MethodPost, ca.ChallengeURL, &reqBody)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "creating request")
	}
	req.SetBasicAuth(ca.Username, ca.Password)
	resp, err := client.Do(req)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "sending request")
	}
	if resp.StatusCode != http.StatusOK {
		return "", ctxerr.Wrap(ctx, fmt.Errorf("status code %d", resp.StatusCode), "getting Smallstep SCEP challenge")
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "reading response body")
	}

	return string(b), nil
}
