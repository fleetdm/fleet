package service

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
	*baseClient
	nodeKeyFilePath string
	enrollSecret    string
	hostInfo        fleet.OrbitHostInfo

	enrolledMu sync.Mutex
	enrolled   bool

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
}

// time-to-live for config cache
const configCacheTTL = 3 * time.Second

type configCache struct {
	mu          sync.Mutex
	lastUpdated time.Time
	config      *fleet.OrbitConfig
	err         error
}

func (oc *OrbitClient) request(verb string, path string, params interface{}, resp interface{}) error {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("making request json marshalling : %w", err)
		}
	}

	parsedURL, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	oc.closeIdleConnections()

	ctx := context.Background()
	if os.Getenv("FLEETD_TEST_HTTPTRACE") == "1" {
		ctx = httptrace.WithClientTrace(ctx, testStdoutHTTPTracer)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		verb,
		oc.url(parsedURL.Path, parsedURL.RawQuery).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}
	oc.setClientCapabilitiesHeader(request)
	response, err := oc.http.Do(request)
	if err != nil {
		oc.setLastRecordedError(err)
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	if err := oc.parseResponse(verb, path, response, resp); err != nil {
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
) (*OrbitClient, error) {
	orbitCapabilities := fleet.GetOrbitClientCapabilities()
	bc, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", fleetClientCert, orbitCapabilities)
	if err != nil {
		return nil, err
	}

	nodeKeyFilePath := filepath.Join(rootDir, constant.OrbitNodeKeyFileName)
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &OrbitClient{
		nodeKeyFilePath:            nodeKeyFilePath,
		baseClient:                 bc,
		enrollSecret:               enrollSecret,
		hostInfo:                   orbitHostInfo,
		enrolled:                   false,
		onGetConfigErrFns:          onGetConfigErrFns,
		lastIdleConnectionsCleanup: time.Now(),
		ReceiverUpdateInterval:     defaultOrbitConfigReceiverInterval,
		receiverUpdateContext:      ctx,
		receiverUpdateCancelFunc:   cancelFunc,
	}, nil
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

// closeIdleConnections attempts to close idle connections from the pool
// every 55 minutes.
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

	c, ok := oc.baseClient.http.(*http.Client)
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
		receiver := receiver
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
			err = oc.authenticatedRequest(verb, path, &orbitGetConfigRequest{}, &resp)
			var (
				netErr        net.Error
				statusCodeErr *statusCodeErr
			)
			if err != nil && oc.onGetConfigErrFns != nil && oc.onGetConfigErrFns.DebugErrFunc != nil {
				oc.onGetConfigErrFns.DebugErrFunc(err)
			}
			if errors.As(err, &netErr) || (errors.As(err, &statusCodeErr) && statusCodeErr.code >= 500) {
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

// SetOrUpdateDeviceToken sends a request to the server to set or update the
// device token with the given value.
func (oc *OrbitClient) SetOrUpdateDeviceToken(deviceAuthToken string) error {
	verb, path := "POST", "/api/fleet/orbit/device_token"
	params := setOrUpdateDeviceTokenRequest{
		DeviceAuthToken: deviceAuthToken,
	}
	var resp setOrUpdateDeviceTokenResponse
	if err := oc.authenticatedRequest(verb, path, &params, &resp); err != nil {
		return err
	}
	return nil
}

// SetOrUpdateDeviceMappingEmail sends a request to the server to set or update the
// device mapping email with the given value.
func (oc *OrbitClient) SetOrUpdateDeviceMappingEmail(email string) error {
	verb, path := "PUT", "/api/fleet/orbit/device_mapping"
	params := orbitPutDeviceMappingRequest{
		Email: email,
	}
	var resp orbitPutDeviceMappingResponse
	if err := oc.authenticatedRequest(verb, path, &params, &resp); err != nil {
		return err
	}
	return nil
}

// GetHostScript returns the script fetched from Fleet server to run on this
// host.
func (oc *OrbitClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/fleet/orbit/scripts/request"
	var resp orbitGetScriptResponse
	if err := oc.authenticatedRequest(verb, path, &orbitGetScriptRequest{
		ExecutionID: execID,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.HostScriptResult, nil
}

// SaveHostScriptResult saves the result of running the script on this host.
func (oc *OrbitClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	verb, path := "POST", "/api/fleet/orbit/scripts/result"
	var resp orbitPostScriptResultResponse
	if err := oc.authenticatedRequest(verb, path, &orbitPostScriptResultRequest{
		HostScriptResultPayload: result,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (oc *OrbitClient) GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error) {
	verb, path := "POST", "/api/fleet/orbit/software_install/details"
	var resp orbitGetSoftwareInstallResponse
	if err := oc.authenticatedRequest(verb, path, &orbitGetSoftwareInstallRequest{
		InstallUUID: installId,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.SoftwareInstallDetails, nil
}

func (oc *OrbitClient) SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error {
	verb, path := "POST", "/api/fleet/orbit/software_install/result"
	var resp orbitPostSoftwareInstallResultResponse
	if err := oc.authenticatedRequest(verb, path, &orbitPostSoftwareInstallResultRequest{
		HostSoftwareInstallResultPayload: payload,
	}, &resp); err != nil {
		return err
	}
	return nil
}

func (oc *OrbitClient) DownloadSoftwareInstaller(installerID uint, downloadDirectory string) (string, error) {
	verb, path := "POST", "/api/fleet/orbit/software_install/package?alt=media"
	resp := FileResponse{DestPath: downloadDirectory}
	if err := oc.authenticatedRequest(verb, path, &orbitDownloadSoftwareInstallerRequest{
		InstallerID: installerID,
	}, &resp); err != nil {
		return "", err
	}
	return resp.GetFilePath(), nil
}

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
	return oc.authenticatedRequest(verb, path, &orbitDownloadSoftwareInstallerRequest{
		InstallerID: installerID,
	}, &resp)
}

// Ping sends a ping request to the orbit/ping endpoint.
func (oc *OrbitClient) Ping() error {
	verb, path := "HEAD", "/api/fleet/orbit/ping"
	err := oc.request(verb, path, nil, nil)
	if err == nil || errors.Is(err, notFoundErr{}) {
		// notFound is ok, it means an old server without the capabilities header
		return nil
	}
	return err
}

func (oc *OrbitClient) enroll() (string, error) {
	verb, path := "POST", "/api/fleet/orbit/enroll"
	params := EnrollOrbitRequest{
		EnrollSecret:      oc.enrollSecret,
		HardwareUUID:      oc.hostInfo.HardwareUUID,
		HardwareSerial:    oc.hostInfo.HardwareSerial,
		Hostname:          oc.hostInfo.Hostname,
		Platform:          oc.hostInfo.Platform,
		OsqueryIdentifier: oc.hostInfo.OsqueryIdentifier,
		ComputerName:      oc.hostInfo.ComputerName,
		HardwareModel:     oc.hostInfo.HardwareModel,
	}
	var resp EnrollOrbitResponse
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
	switch {
	case err == nil:
		return string(orbitNodeKey), nil
	case errors.Is(err, fs.ErrNotExist):
		// OK, if there's no orbit node key, proceed to enroll.
	default:
		return "", fmt.Errorf("read orbit node key file: %w", err)
	}
	var (
		orbitNodeKey_        string
		endpointDoesNotExist bool
	)
	if err := retry.Do(
		func() error {
			var err error
			orbitNodeKey_, err = oc.enrollAndWriteNodeKeyFile()
			switch {
			case err == nil:
				return nil
			case errors.Is(err, notFoundErr{}):
				// Do not retry if the endpoint does not exist.
				endpointDoesNotExist = true
				return nil
			default:
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "enroll failed, retrying")
				return err
			}
		},
		// The below configuration means the following retry intervals (exponential backoff):
		// 10s, 20s, 40s, 80s, 160s and then return the failure (max attempts = 6)
		// thus executing no more than ~6 enroll request failures every ~5 minutes.
		retry.WithInterval(orbitEnrollRetryInterval()),
		retry.WithMaxAttempts(constant.OrbitEnrollMaxRetries),
		retry.WithBackoffMultiplier(constant.OrbitEnrollBackoffMultiplier),
	); err != nil {
		return "", fmt.Errorf("orbit node key enroll failed, attempts=%d", constant.OrbitEnrollMaxRetries)
	}
	if endpointDoesNotExist {
		return "", errors.New("enroll endpoint does not exist")
	}
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

	if runtime.GOOS == "windows" {

		// creating the secret file with empty content
		if err := os.WriteFile(oc.nodeKeyFilePath, nil, constant.DefaultFileMode); err != nil {
			return "", fmt.Errorf("create orbit node key file: %w", err)
		}

		// restricting file access
		if err := platform.ChmodRestrictFile(oc.nodeKeyFilePath); err != nil {
			return "", fmt.Errorf("apply ACLs: %w", err)
		}
	}

	// writing raw key material to the acl-ready secret file
	if err := os.WriteFile(oc.nodeKeyFilePath, []byte(orbitNodeKey), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("write orbit node key file: %w", err)
	}

	return orbitNodeKey, nil
}

func (oc *OrbitClient) authenticatedRequest(verb string, path string, params interface{}, resp interface{}) error {
	nodeKey, err := oc.getNodeKeyOrEnroll()
	if err != nil {
		return err
	}

	s := params.(setOrbitNodeKeyer)
	s.setOrbitNodeKey(nodeKey)

	err = oc.request(verb, path, params, resp)
	switch {
	case err == nil:
		oc.setEnrolled(true)
		return nil
	case errors.Is(err, ErrUnauthenticated):
		if err := os.Remove(oc.nodeKeyFilePath); err != nil {
			log.Info().Err(err).Msg("remove orbit node key")
		}
		oc.setEnrolled(false)
		return err
	default:
		return err
	}
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

// SetOrUpdateDiskEncryptionKey sends a request to the server to set or update the disk
// encryption keys and result of the encryption process
func (oc *OrbitClient) SetOrUpdateDiskEncryptionKey(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error {
	verb, path := "POST", "/api/fleet/orbit/disk_encryption_key"

	var resp orbitPostDiskEncryptionKeyResponse
	if err := oc.authenticatedRequest(verb, path, &orbitPostDiskEncryptionKeyRequest{
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
func (oc *OrbitClient) GetSetupExperienceStatus() (*fleet.SetupExperienceStatusPayload, error) {
	verb, path := "POST", "/api/fleet/orbit/setup_experience/status"
	var resp getOrbitSetupExperienceStatusResponse
	err := oc.authenticatedRequest(verb, path, &getOrbitSetupExperienceStatusRequest{}, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Results, nil
}

func (oc *OrbitClient) SendLinuxKeyEscrowResponse(lr luks.LuksResponse) error {
	verb, path := "POST", "/api/fleet/orbit/luks_data"
	var resp orbitPostLUKSResponse
	if err := oc.authenticatedRequest(verb, path, &orbitPostLUKSRequest{
		Passphrase:  lr.Passphrase,
		KeySlot:     lr.KeySlot,
		Salt:        lr.Salt,
		ClientError: lr.Err,
	}, &resp); err != nil {
		return err
	}

	return nil
}
