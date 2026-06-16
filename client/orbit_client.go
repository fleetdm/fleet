package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/logging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/luks"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// OrbitClient exposes the Orbit API to communicate with the Fleet server.
type OrbitClient struct {
	*BaseClient
	nodeKeyFilePath string
	enrollSecret    string
	hostInfo        fleet.OrbitHostInfo

	enrolledMu sync.Mutex
	enrolled   bool

	// reenrollMu guards the 401 debounce state below.
	reenrollMu sync.Mutex
	// unauthenticatedSince is when authenticated requests first started failing with a 401 in the current streak. Zero means the most
	// recent authenticated request succeeded (or none have failed). Used to debounce re-enrollment so a transient/spurious 401 does
	// not throw away an otherwise-valid node key.
	unauthenticatedSince time.Time
	// forceReenroll is armed once 401s have persisted past unauthenticatedReenrollGracePeriod. When set, getNodeKeyOrEnroll
	// re-enrolls (overwriting the existing key).
	forceReenroll bool

	lastRecordedErrMu sync.Mutex
	lastRecordedErr   error

	configCache                 configCache
	onGetConfigErrFns           *OnGetConfigErrFuncs
	lastNetErrOnGetConfigLogged time.Time

	lastIdleConnectionsCleanupMu sync.Mutex
	lastIdleConnectionsCleanup   time.Time

	// TestNodeKey is used for testing only.
	TestNodeKey string

	// Interfaces that will receive updated configs
	ConfigReceivers []fleet.OrbitConfigReceiver
	// How frequently a new config will be fetched
	ReceiverUpdateInterval time.Duration
	// receiverUpdateContext used by ExecuteConfigReceivers to cancel the update loop.
	receiverUpdateContext context.Context
	// receiverUpdateCancelFunc is used to cancel receiverUpdateContext.
	receiverUpdateCancelFunc context.CancelFunc

	// euaToken is a one-time Fleet-signed JWT from Windows MDM enrollment,
	// sent during orbit enrollment to link the IdP account without prompting.
	euaToken string

	// hostIdentityCertPath is the file path to the host identity certificate issued using SCEP.
	//
	// If set, it is deleted once HTTP 401 errors from Fleet have persisted past unauthenticatedReenrollGracePeriod (see
	// authenticatedRequest), which also causes ExecuteConfigReceivers to terminate and trigger a restart.
	hostIdentityCertPath string

	// lastSSOWindowOpen tracks when the SSO browser window was last opened.
	// If zero, no window has been opened yet. Used to periodically re-open
	// the SSO window if the user closes it before completing authentication.
	lastSSOWindowOpen time.Time

	// openSSOWindow is a function that opens a browser window to the SSO URL.
	openSSOWindow func() error
}

// time-to-live for config cache
const configCacheTTL = 3 * time.Second

// ssoWindowReopenInterval is the minimum time between SSO window open attempts.
// If the user closes the SSO browser window before completing authentication,
// the window will be re-opened after this interval.
const ssoWindowReopenInterval = 5 * time.Minute

// unauthenticatedReenrollGracePeriod is how long authenticated requests must keep failing with a 401 before orbit re-enrolls.
// It is a var (not a const) so tests can shorten it.
var unauthenticatedReenrollGracePeriod = 45 * time.Second

type configCache struct {
	mu          sync.Mutex
	lastUpdated time.Time
	config      *fleet.OrbitConfig
	err         error
}

func (oc *OrbitClient) SetOpenSSOWindowFunc(f func() error) {
	oc.openSSOWindow = f
}

func (oc *OrbitClient) request(verb string, path string, params any, resp any) error {
	return oc.requestWithExternal(verb, path, params, resp, false)
}

// requestWithExternal is used to make requests to Fleet or external URLs. If external is true, the pathOrURL
// is used as the full URL to make the request to.
func (oc *OrbitClient) requestWithExternal(verb string, pathOrURL string, params any, resp any, external bool) error {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("making request json marshalling : %w", err)
		}
	}

	oc.closeIdleConnections()

	ctx := context.Background()
	if os.Getenv("FLEETD_TEST_HTTPTRACE") == "1" {
		ctx = httptrace.WithClientTrace(ctx, testStdoutHTTPTracer)
	}

	var request *http.Request
	if external {
		request, err = http.NewRequestWithContext(
			ctx,
			verb,
			pathOrURL,
			nil,
		)
		if err != nil {
			return err
		}
	} else {
		parsedURL, err := url.Parse(pathOrURL)
		if err != nil {
			return fmt.Errorf("parsing URL: %w", err)
		}

		request, err = http.NewRequestWithContext(
			ctx,
			verb,
			oc.URL(parsedURL.Path, parsedURL.RawQuery).String(),
			bytes.NewBuffer(bodyBytes),
		)
		if err != nil {
			return err
		}
		oc.SetClientCapabilitiesHeader(request)
	}
	response, err := oc.DoHTTPRequest(request)
	if err != nil {
		oc.setLastRecordedError(err)
		return fmt.Errorf("%s %s: %w", verb, pathOrURL, err)
	}
	defer response.Body.Close()

	if err := oc.ParseResponse(verb, pathOrURL, response, resp); err != nil {
		oc.setLastRecordedError(err)
		return err
	}
	return nil
}

// OnGetConfigErrFuncs defines functions to be executed on GetConfig errors.
type OnGetConfigErrFuncs struct {
	// OnNetErrFunc receives network and 5XX errors on GetConfig requests.
	// These errors are rate limited to once every 5 minutes.
	OnNetErrFunc func(err error)
	// DebugErrFunc receives all errors on GetConfig requests.
	DebugErrFunc func(err error)
}

var (
	netErrInterval                     = 5 * time.Minute
	configRetryOnNetworkError          = 30 * time.Second
	defaultOrbitConfigReceiverInterval = 30 * time.Second
)

// NewOrbitClient creates a new OrbitClient.
//
//   - rootDir is the Orbit's root directory, where the Orbit node key is loaded-from/stored.
//   - addr is the address of the Fleet server.
//   - orbitHostInfo is the host system information used for enrolling to Fleet.
//   - onGetConfigErrFns can be used to handle errors in the GetConfig request.
func NewOrbitClient(
	rootDir string,
	addr string,
	rootCA string,
	insecureSkipVerify bool,
	enrollSecret string,
	fleetClientCert *tls.Certificate,
	orbitHostInfo fleet.OrbitHostInfo,
	onGetConfigErrFns *OnGetConfigErrFuncs,
	httpSignerWrapper func(*http.Client) *http.Client,
	hostIdentityCertPath string,
) (*OrbitClient, error) {
	orbitCapabilities := fleet.GetOrbitClientCapabilities()
	bc, err := NewBaseClient(addr, insecureSkipVerify, rootCA, "", fleetClientCert, orbitCapabilities, httpSignerWrapper)
	if err != nil {
		return nil, err
	}

	nodeKeyFilePath := filepath.Join(rootDir, constant.OrbitNodeKeyFileName)
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &OrbitClient{
		nodeKeyFilePath:            nodeKeyFilePath,
		BaseClient:                 bc,
		enrollSecret:               enrollSecret,
		hostInfo:                   orbitHostInfo,
		enrolled:                   false,
		onGetConfigErrFns:          onGetConfigErrFns,
		lastIdleConnectionsCleanup: time.Now(),
		ReceiverUpdateInterval:     defaultOrbitConfigReceiverInterval,
		receiverUpdateContext:      ctx,
		receiverUpdateCancelFunc:   cancelFunc,
		hostIdentityCertPath:       hostIdentityCertPath,
	}, nil
}

// SetEUAToken sets a one-time EUA token to include in the enrollment request.
func (oc *OrbitClient) SetEUAToken(token string) {
	oc.euaToken = token
}

// TriggerOrbitRestart triggers a orbit process restart.
func (oc *OrbitClient) TriggerOrbitRestart(reason string) {
	log.Info().Msgf("orbit restart triggered: %s", reason)
	oc.receiverUpdateCancelFunc()
}

// RestartTriggered returns true if any of the config receivers triggered an orbit restart.
func (oc *OrbitClient) RestartTriggered() bool {
	select {
	case <-oc.receiverUpdateContext.Done():
		return true
	default:
		return false
	}
}

// closeIdleConnections attempts to close idle connections from the pool every 55 minutes.
//
// Some load balancers (e.g. AWS ELB) have a maximum lifetime for a connection
// (no matter if the connection is active or not) and will forcefully close the
// connection causing errors in the client (e.g. https://github.com/fleetdm/fleet/issues/18783).
// To prevent these errors, we will attempt to cleanup idle connections every 55
// minutes to not let these connection grow too old. (AWS ELB's default value for maximum
// lifetime of a connection is 3600 seconds.)
func (oc *OrbitClient) closeIdleConnections() {
	oc.lastIdleConnectionsCleanupMu.Lock()
	defer oc.lastIdleConnectionsCleanupMu.Unlock()

	if time.Since(oc.lastIdleConnectionsCleanup) < 55*time.Minute {
		return
	}

	oc.lastIdleConnectionsCleanup = time.Now()

	rawClient := oc.GetRawHTTPClient()
	c, ok := rawClient.(*http.Client)
	if !ok {
		return
	}
	t, ok := c.Transport.(*http.Transport)
	if !ok {
		return
	}

	t.CloseIdleConnections()
}

func (oc *OrbitClient) RunConfigReceivers() error {
	config, err := oc.GetConfig()
	if err != nil {
		return fmt.Errorf("RunConfigReceivers get config: %w", err)
	}

	var errs []error
	var errMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(oc.ConfigReceivers))

	for _, receiver := range oc.ConfigReceivers {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					errMu.Lock()
					errs = append(errs, fmt.Errorf("panic occured in receiver: %v", err))
					errMu.Unlock()
				}
				wg.Done()
			}()

			err := receiver.Run(config)
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (oc *OrbitClient) RegisterConfigReceiver(cr fleet.OrbitConfigReceiver) {
	oc.ConfigReceivers = append(oc.ConfigReceivers, cr)
}

func (oc *OrbitClient) ExecuteConfigReceivers() error {
	ticker := time.NewTicker(oc.ReceiverUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-oc.receiverUpdateContext.Done():
			return nil
		case <-ticker.C:
			if err := oc.RunConfigReceivers(); err != nil {
				log.Error().Err(err).Msg("running config receivers")
			}
		}
	}
}

func (oc *OrbitClient) InterruptConfigReceivers(err error) {
	oc.receiverUpdateCancelFunc()
}

// GetConfig returns the Orbit config fetched from Fleet server for this instance of OrbitClient.
// Since this method is called in multiple places, we use a cache with configCacheTTL time-to-live
// to reduce traffic to the Fleet server.
// Upon network errors, this method will retry the get config request (every 30 seconds).
func (oc *OrbitClient) GetConfig() (*fleet.OrbitConfig, error) {
	oc.configCache.mu.Lock()
	defer oc.configCache.mu.Unlock()

	// If time-to-live passed, we update the config cache
	now := time.Now()
	if now.After(oc.configCache.lastUpdated.Add(configCacheTTL)) {
		verb, path := "POST", "/api/fleet/orbit/config"
		var (
			resp fleet.OrbitConfig
			err  error
		)
		// Retry until we don't get a network error or a 5XX error.
		_ = retry.Do(func() error {
			err = oc.authenticatedRequest(verb, path, &fleet.OrbitGetConfigRequest{}, &resp)
			var (
				netErr        net.Error
				statusCodeErr *StatusCodeErr
			)
			if err != nil && oc.onGetConfigErrFns != nil && oc.onGetConfigErrFns.DebugErrFunc != nil {
				oc.onGetConfigErrFns.DebugErrFunc(err)
			}
			if errors.As(err, &netErr) || (errors.As(err, &statusCodeErr) && statusCodeErr.StatusCode() >= 500) {
				now := time.Now()
				if oc.onGetConfigErrFns != nil && oc.onGetConfigErrFns.OnNetErrFunc != nil && now.After(oc.lastNetErrOnGetConfigLogged.Add(netErrInterval)) {
					oc.onGetConfigErrFns.OnNetErrFunc(err)
					oc.lastNetErrOnGetConfigLogged = now
				}
				return err // retry on network or server 5XX errors
			}
			return nil
		}, retry.WithInterval(configRetryOnNetworkError))
		oc.configCache.config = &resp
		oc.configCache.err = err
		oc.configCache.lastUpdated = now
	}
	return oc.configCache.config, oc.configCache.err
}

// SetOrUpdateDeviceToken sends a request to the server to set or update the device token.
func (oc *OrbitClient) SetOrUpdateDeviceToken(deviceAuthToken string) error {
	verb, path := "POST", "/api/fleet/orbit/device_token"
	params := fleet.SetOrUpdateDeviceTokenRequest{
		DeviceAuthToken: deviceAuthToken,
	}
	var resp fleet.SetOrUpdateDeviceTokenResponse
	if err := oc.authenticatedRequest(verb, path, &params, &resp); err != nil {
		return err
	}
	return nil
}

// SetOrUpdateDeviceMappingEmail sends a request to the server to set or update the device mapping email.
func (oc *OrbitClient) SetOrUpdateDeviceMappingEmail(email string) error {
	verb, path := "PUT", "/api/fleet/orbit/device_mapping"
	params := fleet.OrbitPutDeviceMappingRequest{
		Email: email,
	}
	var resp fleet.OrbitPutDeviceMappingResponse
	if err := oc.authenticatedRequest(verb, path, &params, &resp); err != nil {
		return err
	}
	return nil
}

// GetHostScript returns the script fetched from Fleet server to run on this host.
func (oc *OrbitClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/fleet/orbit/scripts/request"
	var resp fleet.OrbitGetScriptResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitGetScriptRequest{
		ExecutionID: execID,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.HostScriptResult, nil
}

// SaveHostScriptResult saves the result of running the script on this host.
func (oc *OrbitClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	verb, path := "POST", "/api/fleet/orbit/scripts/result"
	var resp fleet.OrbitPostScriptResultResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitPostScriptResultRequest{
		HostScriptResultPayload: result,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (oc *OrbitClient) GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error) {
	verb, path := "POST", "/api/fleet/orbit/software_install/details"
	var resp fleet.OrbitGetSoftwareInstallResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitGetSoftwareInstallRequest{
		InstallUUID: installId,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.SoftwareInstallDetails, nil
}

func (oc *OrbitClient) SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error {
	verb, path := "POST", "/api/fleet/orbit/software_install/result"
	var resp fleet.OrbitPostSoftwareInstallResultResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitPostSoftwareInstallResultRequest{
		HostSoftwareInstallResultPayload: payload,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (oc *OrbitClient) DownloadSoftwareInstaller(installerID uint, downloadDirectory string, progressFunc func(n int)) (string, error) {
	verb, path := "POST", "/api/fleet/orbit/software_install/package?alt=media"
	resp := FileResponse{
		DestPath:     downloadDirectory,
		ProgressFunc: progressFunc,
	}
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitDownloadSoftwareInstallerRequest{
		InstallerID: installerID,
	}, &resp); err != nil {
		return "", err
	}
	return resp.GetFilePath(), nil
}

func (oc *OrbitClient) DownloadSoftwareInstallerFromURL(url string, filename string, downloadDirectory string, progressFunc func(int)) (string, error) {
	resp := FileResponse{
		DestPath:      downloadDirectory,
		DestFile:      filename,
		SkipMediaType: true,
		ProgressFunc:  progressFunc,
	}
	if err := oc.requestWithExternal("GET", url, nil, &resp, true); err != nil {
		return "", err
	}
	return resp.GetFilePath(), nil
}

// NullFileResponse discards downloaded file content.
type NullFileResponse struct{}

func (f *NullFileResponse) Handle(resp *http.Response) error {
	_, _, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return fmt.Errorf("parsing media type from response header: %w", err)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("copying from http stream to io.Discard: %w", err)
	}
	return nil
}

// DownloadAndDiscardSoftwareInstaller downloads the software installer and discards it.
// This method is used during load testing by osquery-perf.
func (oc *OrbitClient) DownloadAndDiscardSoftwareInstaller(installerID uint) error {
	verb, path := "POST", "/api/fleet/orbit/software_install/package?alt=media"
	resp := NullFileResponse{}
	return oc.authenticatedRequest(verb, path, &fleet.OrbitDownloadSoftwareInstallerRequest{
		InstallerID: installerID,
	}, &resp)
}

// Ping sends a ping request to the orbit/ping endpoint.
func (oc *OrbitClient) Ping() error {
	verb, path := "HEAD", "/api/fleet/orbit/ping"
	err := oc.request(verb, path, nil, nil)
	if err == nil || IsNotFoundErr(err) {
		// notFound is ok, it means an old server without the capabilities header
		return nil
	}
	return err
}

func (oc *OrbitClient) enroll() (string, error) {
	verb, path := "POST", "/api/fleet/orbit/enroll"
	params := fleet.EnrollOrbitRequest{
		EnrollSecret:      oc.enrollSecret,
		HardwareUUID:      oc.hostInfo.HardwareUUID,
		HardwareSerial:    oc.hostInfo.HardwareSerial,
		Hostname:          oc.hostInfo.Hostname,
		Platform:          oc.hostInfo.Platform,
		PlatformLike:      oc.hostInfo.PlatformLike,
		OsqueryIdentifier: oc.hostInfo.OsqueryIdentifier,
		ComputerName:      oc.hostInfo.ComputerName,
		HardwareModel:     oc.hostInfo.HardwareModel,
		EUAToken:          oc.euaToken,
	}
	var resp fleet.EnrollOrbitResponse
	err := oc.request(verb, path, params, &resp)
	if err != nil {
		return "", err
	}
	return resp.OrbitNodeKey, nil
}

// enrollLock helps protect the enrolling process in case mutliple OrbitClients
// want to re-enroll at the same time.
var enrollLock sync.Mutex

// getNodeKeyOrEnroll attempts to read the orbit node key if the file exists on disk
// otherwise it enrolls the host with Fleet and saves the node key to disk
func (oc *OrbitClient) getNodeKeyOrEnroll() (string, error) {
	if oc.TestNodeKey != "" {
		return oc.TestNodeKey, nil
	}

	enrollLock.Lock()
	defer enrollLock.Unlock()

	orbitNodeKey, err := os.ReadFile(oc.nodeKeyFilePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("read orbit node key file: %w", err)
	}
	// A present, non-empty key is reused unless the server has been rejecting it (forceReenroll). Read it once so the reuse check and
	// the log message below agree even if a concurrent 401 flips it.
	forced := oc.reenrollForced()
	if err == nil && len(bytes.TrimSpace(orbitNodeKey)) > 0 && !forced {
		return string(orbitNodeKey), nil
	}

	switch {
	case forced:
		// The node key on disk was repeatedly rejected with a 401; re-enroll to obtain a fresh one.
		log.Info().Str("server", oc.BaseURL.String()).Msg("orbit node key was rejected by the server, re-enrolling")
	case err == nil:
		// The file exists but is empty (e.g. a prior write was interrupted). Enroll instead of
		// sending an empty key that the server would reject with a 401.
		log.Info().Str("server", oc.BaseURL.String()).Msg("orbit node key file is empty, enrolling")
	default:
		log.Info().Str("server", oc.BaseURL.String()).Msg("orbit node key file not found, enrolling")
	}

	var orbitNodeKey_ string
	if err := retry.Do(
		func() error {
			orbitNodeKey_, err = oc.enrollAndWriteNodeKeyFile()
			return err
		},
		// The below configuration means the following retry intervals (exponential backoff):
		// 10s, 20s, 40s, 80s, 160s and then return the failure (max attempts = 6)
		// thus executing no more than ~6 enroll request failures every ~5 minutes.
		retry.WithInterval(orbitEnrollRetryInterval()),
		retry.WithMaxAttempts(constant.OrbitEnrollMaxRetries),
		retry.WithBackoffMultiplier(constant.OrbitEnrollBackoffMultiplier),
		retry.WithErrorFilter(func(err error) (errorOutcome retry.ErrorOutcome) {
			log.Info().Err(err).Msg("orbit enroll attempt failed")
			switch {
			case IsNotFoundErr(err):
				// Do not retry if the endpoint does not exist.
				return retry.ErrorOutcomeDoNotRetry
			case errors.Is(err, ErrEndUserAuthRequired):
				// If we get an ErrEndUserAuthRequired error, then the user
				// needs to authenticate with the identity provider.
				//
				// Open a browser window to the sign-on page and
				// then keep retrying until they authenticate.
				log.Debug().Msg("enroll unauthenticated, waiting for end-user to authenticate via SSO")
				if oc.lastSSOWindowOpen.IsZero() || time.Since(oc.lastSSOWindowOpen) >= ssoWindowReopenInterval {
					if oc.openSSOWindow == nil {
						log.Error().Msg("SSO window open function not set")
						return retry.ErrorOutcomeNormalRetry
					}
					log.Debug().Msg("opening SSO window")
					openWindowErr := oc.openSSOWindow()
					if openWindowErr != nil {
						log.Error().Err(openWindowErr).Msg("opening SSO window")
						return retry.ErrorOutcomeNormalRetry
					}
					oc.lastSSOWindowOpen = time.Now()
				}
				// Sleep for 20 seconds, making the total retry interval 30 seconds
				time.Sleep(20 * time.Second)
				return retry.ErrorOutcomeResetAttempts
			default:
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "enroll failed, retrying")
				return retry.ErrorOutcomeNormalRetry
			}
		}),
	); err != nil {
		if IsNotFoundErr(err) {
			return "", errors.New("enroll endpoint does not exist")
		}
		return "", fmt.Errorf("orbit node key enroll failed, attempts=%d", constant.OrbitEnrollMaxRetries)
	}
	// Enrollment succeeded and the new key has been written, so clear any armed re-enroll / 401 streak.
	oc.clearReenrollState()
	return orbitNodeKey_, nil
}

// GetNodeKey gets the orbit node key from file.
func (oc *OrbitClient) GetNodeKey() (string, error) {
	orbitNodeKey, err := os.ReadFile(oc.nodeKeyFilePath)
	if err != nil {
		return "", err
	}
	return string(orbitNodeKey), nil
}

func (oc *OrbitClient) enrollAndWriteNodeKeyFile() (string, error) {
	orbitNodeKey, err := oc.enroll()
	if err != nil {
		return "", fmt.Errorf("enroll request: %w", err)
	}

	// Write the new node key atomically: write+restrict a temp file in the same directory, then rename it over the destination. This
	// guarantees we never truncate or remove an existing, still-valid node key until the new key is fully on disk, so a crash
	// mid-write (or an enroll that is ultimately rejected) cannot leave the host with an empty or missing node key file.
	tmp, err := os.CreateTemp(filepath.Dir(oc.nodeKeyFilePath), ".orbit-node-key-*")
	if err != nil {
		return "", fmt.Errorf("create temp orbit node key file: %w", err)
	}
	tmpPath := tmp.Name()
	renamed := false
	closed := false
	closeTmp := func() error {
		if closed {
			return nil
		}
		closed = true
		return tmp.Close()
	}
	defer func() {
		if !renamed {
			_ = closeTmp()
			_ = os.Remove(tmpPath)
		}
	}()

	if runtime.GOOS == "windows" {
		// Restrict file access before the key material is written so the secret is never on disk
		// with default (inherited) permissions.
		if err := platform.ChmodRestrictFile(tmpPath); err != nil {
			return "", fmt.Errorf("apply ACLs: %w", err)
		}
	}
	if _, err := tmp.WriteString(orbitNodeKey); err != nil {
		return "", fmt.Errorf("write temp orbit node key file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("sync temp orbit node key file: %w", err)
	}
	if err := closeTmp(); err != nil {
		return "", fmt.Errorf("close temp orbit node key file: %w", err)
	}
	// os.Rename atomically replaces the destination; this matches orbit's existing atomic-write pattern (e.g. pkg/update). On Windows
	// a rename can fail with a sharing violation if another process (e.g. antivirus/EDR) holds the destination open at this instant
	// (orbit already contends with Windows file-in-use locking elsewhere, see orbit/pkg/platform). On failure we don't leave an empty
	// or partial file: the prior key stays on disk. enroll() has already rotated the server-side key by this point, so that on-disk
	// key is now stale and the host will 401 and re-enroll via the debounce path on a later attempt (self-healing).
	if err := os.Rename(tmpPath, oc.nodeKeyFilePath); err != nil {
		return "", fmt.Errorf("replace orbit node key file: %w", err)
	}
	renamed = true

	return orbitNodeKey, nil
}

func (oc *OrbitClient) authenticatedRequest(verb string, path string, params any, resp any) error {
	nodeKey, err := oc.getNodeKeyOrEnroll()
	if err != nil {
		return err
	}

	s := params.(fleet.SetOrbitNodeKeyer)
	s.SetOrbitNodeKey(nodeKey)

	err = oc.request(verb, path, params, resp)
	switch {
	case err == nil:
		oc.setEnrolled(true)
		// A successful authenticated request means the node key is valid; clear any 401 streak.
		oc.clearReenrollState()
		return nil
	case errors.Is(err, ErrUnauthenticated):
		// A 401 on an authenticated request means the server rejected the orbit node key. Rather than reacting to a single 401, we wait until 401s have persisted for unauthenticatedReenrollGracePeriod.
		oc.setEnrolled(false)
		reenroll, waited := oc.noteUnauthenticated()
		if reenroll {
			log.Info().Str("path", path).Str("after", waited.Round(time.Second).String()).
				Msg("orbit node key repeatedly rejected with 401, will re-enroll on next request")
		} else {
			log.Info().Str("path", path).Str("after", waited.Round(time.Second).String()).
				Msg("orbit received 401 unauthenticated, retrying with existing node key before re-enrolling")
			return err
		}

		if oc.hostIdentityCertPath != "" {
			if err := os.Remove(oc.hostIdentityCertPath); err != nil {
				log.Info().Err(err).Msg("remove orbit host identity cert")
			}
			log.Info().Msg("removed orbit host identity cert, triggering a restart")
			oc.receiverUpdateCancelFunc()
		}
		return err
	default:
		return err
	}
}

// noteUnauthenticated records a 401 on an authenticated request and reports whether 401s have persisted long enough
// (unauthenticatedReenrollGracePeriod) to warrant a re-enroll, along with how long the current 401 streak has lasted. When it
// returns true it arms forceReenroll, which getNodeKeyOrEnroll acts on (overwriting the existing key) on the next call.
func (oc *OrbitClient) noteUnauthenticated() (reenroll bool, waited time.Duration) {
	oc.reenrollMu.Lock()
	defer oc.reenrollMu.Unlock()
	now := time.Now()
	if oc.unauthenticatedSince.IsZero() {
		oc.unauthenticatedSince = now
	}
	waited = now.Sub(oc.unauthenticatedSince)
	if waited >= unauthenticatedReenrollGracePeriod {
		oc.forceReenroll = true
		return true, waited
	}
	return false, waited
}

// reenrollForced reports whether a re-enroll has been armed by repeated 401s.
func (oc *OrbitClient) reenrollForced() bool {
	oc.reenrollMu.Lock()
	defer oc.reenrollMu.Unlock()
	return oc.forceReenroll
}

// clearReenrollState resets the 401 streak and any armed re-enroll, called after a successful
// authenticated request or a successful (re-)enroll.
func (oc *OrbitClient) clearReenrollState() {
	oc.reenrollMu.Lock()
	defer oc.reenrollMu.Unlock()
	oc.unauthenticatedSince = time.Time{}
	oc.forceReenroll = false
}

func (oc *OrbitClient) Enrolled() bool {
	oc.enrolledMu.Lock()
	defer oc.enrolledMu.Unlock()
	return oc.enrolled
}

func (oc *OrbitClient) setEnrolled(v bool) {
	oc.enrolledMu.Lock()
	defer oc.enrolledMu.Unlock()
	oc.enrolled = v
}

func (oc *OrbitClient) LastRecordedError() error {
	oc.lastRecordedErrMu.Lock()
	defer oc.lastRecordedErrMu.Unlock()
	return oc.lastRecordedErr
}

func (oc *OrbitClient) setLastRecordedError(err error) {
	oc.lastRecordedErrMu.Lock()
	defer oc.lastRecordedErrMu.Unlock()
	oc.lastRecordedErr = fmt.Errorf("%s: %w", time.Now().UTC().Format("2006-01-02T15:04:05Z"), err)
}

func orbitEnrollRetryInterval() time.Duration {
	interval := os.Getenv("FLEETD_ENROLL_RETRY_INTERVAL")
	if interval != "" {
		d, err := time.ParseDuration(interval)
		if err == nil {
			return d
		}
	}
	return constant.OrbitEnrollRetrySleep
}

// SetOrUpdateDiskEncryptionKey sends a request to the server to set or update the disk encryption keys.
func (oc *OrbitClient) SetOrUpdateDiskEncryptionKey(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error {
	verb, path := "POST", "/api/fleet/orbit/disk_encryption_key"
	var resp fleet.OrbitPostDiskEncryptionKeyResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitPostDiskEncryptionKeyRequest{
		EncryptionKey: diskEncryptionStatus.EncryptionKey,
		ClientError:   diskEncryptionStatus.ClientError,
	}, &resp); err != nil {
		return err
	}
	return nil
}

const httpTraceTimeFormat = "2006-01-02T15:04:05Z"

var testStdoutHTTPTracer = &httptrace.ClientTrace{
	ConnectStart: func(network, addr string) {
		fmt.Printf(
			"httptrace: %s: ConnectStart: %s, %s\n",
			time.Now().UTC().Format(httpTraceTimeFormat), network, addr,
		)
	},
	ConnectDone: func(network, addr string, err error) {
		fmt.Printf(
			"httptrace: %s: ConnectDone: %s, %s, err='%s'\n",
			time.Now().UTC().Format(httpTraceTimeFormat), network, addr, err,
		)
	},
}

// GetSetupExperienceStatus checks the status of the setup experience for this host.
func (oc *OrbitClient) GetSetupExperienceStatus(resetFailedSetupSteps bool) (*fleet.SetupExperienceStatusPayload, error) {
	verb, path := "POST", "/api/fleet/orbit/setup_experience/status"
	var resp fleet.GetOrbitSetupExperienceStatusResponse
	err := oc.authenticatedRequest(verb, path, &fleet.GetOrbitSetupExperienceStatusRequest{
		ResetFailedSetupSteps: resetFailedSetupSteps,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (oc *OrbitClient) SendLinuxKeyEscrowResponse(lr luks.LuksResponse) error {
	verb, path := "POST", "/api/fleet/orbit/luks_data"
	var resp fleet.OrbitPostLUKSResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitPostLUKSRequest{
		Passphrase:  lr.Passphrase,
		KeySlot:     lr.KeySlot,
		Salt:        lr.Salt,
		ClientError: lr.Err,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (oc *OrbitClient) InitiateSetupExperience() (fleet.SetupExperienceInitResult, error) {
	verb, path := "POST", "/api/fleet/orbit/setup_experience/init"
	var resp fleet.OrbitSetupExperienceInitResponse
	if err := oc.authenticatedRequest(verb, path, &fleet.OrbitSetupExperienceInitRequest{}, &resp); err != nil {
		return fleet.SetupExperienceInitResult{}, err
	}
	return resp.Result, nil
}
