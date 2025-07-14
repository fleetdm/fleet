package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	_              scepserver.ServiceWithIdentifier = (*scepProxyService)(nil)
	challengeRegex                                  = regexp.MustCompile(`(?i)The enrollment challenge password is: <B> (?P<password>\S*)`)
)

const (
	fullPasswordCache             = "The password cache is full."
	ndesInsufficientPermissions   = "You do not have sufficient permission to enroll with SCEP."
	MessageSCEPProxyNotConfigured = "SCEP proxy is not configured"
	NDESChallengeInvalidAfter     = 57 * time.Minute
)

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

	client, err := scepclient.New(scepURL, svc.debugLogger, svc.Timeout, false)
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

	client, err := scepclient.New(scepURL, svc.debugLogger, svc.Timeout, false)
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

	client, err := scepclient.New(scepURL, svc.debugLogger, svc.Timeout, false)
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
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting app config")
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
	if !strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix) {
		return "", &scepserver.BadRequestError{Message: fmt.Sprintf("invalid profile UUID (only Apple config profiles are supported): %s",
			profileUUID)}
	}
	profile, err := svc.ds.GetHostMDMCertificateProfile(ctx, hostUUID, profileUUID, caName)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting host MDM profile")
	}
	if profile == nil {
		// Return error that implements kithttp.StatusCoder interface
		return "", &scepserver.BadRequestError{Message: "unknown identifier in URL path"}
	}
	if profile.Status == nil || *profile.Status != fleet.MDMDeliveryPending {
		// This could happen if Fleet DB was updated before the profile was updated on the host.
		// We expect another certificate request from the host once the profile is updated.
		status := "null"
		if profile.Status != nil {
			status = string(*profile.Status)
		}
		return "", &scepserver.BadRequestError{Message: fmt.Sprintf("profile status (%s) is not 'pending' for host:%s profile:%s", status,
			hostUUID, profileUUID)}
	}
	var scepURL string
	switch profile.Type {
	case fleet.CAConfigNDES:
		if !appConfig.Integrations.NDESSCEPProxy.Valid {
			// Return error that implements kithttp.StatusCoder interface
			return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
		}
		if checkChallenge && profile.ChallengeRetrievedAt != nil && profile.ChallengeRetrievedAt.Add(NDESChallengeInvalidAfter).Before(time.Now()) {
			// The challenge password was retrieved for this profile, and is now invalid.
			// We need to resend the profile with a new challenge password.
			// Note: we don't actually know if it is invalid, and we can't get that exact feedback from SCEP server.
			if err = svc.ds.ResendHostMDMProfile(ctx, hostUUID, profileUUID); err != nil {
				return "", ctxerr.Wrap(ctx, err, "resending host mdm profile")
			}
			return "", &scepserver.BadRequestError{Message: "challenge password has expired"}
		}
		scepURL = appConfig.Integrations.NDESSCEPProxy.Value.URL
	case fleet.CAConfigCustomSCEPProxy:
		if !appConfig.Integrations.CustomSCEPProxy.Valid {
			return "", &scepserver.BadRequestError{Message: MessageSCEPProxyNotConfigured}
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
		for _, ca := range appConfig.Integrations.CustomSCEPProxy.Value {
			if ca.Name == profile.CAName {
				scepURL = ca.URL
				break
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
	var errs []error

	if err := svc.ds.ConsumeChallenge(ctx, fleetChallenge); err != nil {
		errs = append(errs, ctxerr.Wrap(ctx, err, "custom scep proxy: validating challenge"))
		// FIXME: We really should have a more generic function to handle this, but our existing methods
		// for "resending" profiles don't reevaluate the profile variables so they aren't useful for
		// custom SCEP profiles where we need to regenerate the SCEP challenge. The main difference between
		// the existing flow and the implementation below is that we need to blank the command uuid in order
		// get the reconcile cron to reevaluate the command template to generate the challenge. Otherwise,
		// it just sends the old bytes again. It feels like we some leaky abstrations somewhere that we need
		// to clean up.
		if err := svc.ds.ResendHostCustomSCEPProfile(ctx, hostUUID, profileUUID); err != nil {
			errs = append(errs, ctxerr.Wrap(ctx, err, "custom scep proxy: resending host mdm profile"))
		}
	}

	if len(errs) > 0 {
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

func (s *SCEPConfigService) ValidateNDESSCEPAdminURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) error {
	_, err := s.GetNDESSCEPChallenge(ctx, proxy)
	return err
}

func (s *SCEPConfigService) GetNDESSCEPChallenge(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) (string, error) {
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
	// Make a transformer that converts MS-Win default to UTF8:
	win16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	// Make a transformer that is like win16le, but abides by BOM:
	utf16bom := unicode.BOMOverride(win16le.NewDecoder())

	// Make a Reader that uses utf16bom:
	unicodeReader := transform.NewReader(resp.Body, utf16bom)
	bodyText, err := io.ReadAll(unicodeReader)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "reading response body")
	}
	htmlString := string(bodyText)

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
	client, err := scepclient.New(url, s.logger, s.Timeout, false)
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
