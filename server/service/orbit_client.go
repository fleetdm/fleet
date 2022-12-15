package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// OrbitClient exposes the Orbit API to communicate with the Fleet server.
type OrbitClient struct {
	*baseClient
	nodeKeyFilePath string
	enrollSecret    string
	uuid            string

	enrolledMu sync.Mutex
	enrolled   bool

	lastRecordedErrMu sync.Mutex
	lastRecordedErr   error

	// TestNodeKey is used for testing only.
	TestNodeKey string
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

	request, err := http.NewRequest(
		verb,
		oc.url(path, "").String(),
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

// NewOrbitClient creates a new OrbitClient.
//
// rootDir is the Orbit's root directory, where the Orbit node key is loaded-from/stored.
// addr is the address of the Fleet server.
// uuid is the UUID of the OrbitClient instance.
func NewOrbitClient(rootDir string, addr string, rootCA string, insecureSkipVerify bool, enrollSecret, uuid string) (*OrbitClient, error) {
	orbitCapabilities := fleet.CapabilityMap{}
	bc, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", orbitCapabilities)
	if err != nil {
		return nil, err
	}

	nodeKeyFilePath := filepath.Join(rootDir, constant.OrbitNodeKeyFileName)
	return &OrbitClient{
		nodeKeyFilePath: nodeKeyFilePath,
		baseClient:      bc,
		enrollSecret:    enrollSecret,
		uuid:            uuid,
		enrolled:        false,
	}, nil
}

// OrbitConfig holds the config returned by the Fleet server for a Orbit instance.
type OrbitConfig struct {
	// Flags holds the osquery startup flags to use when running osquery.
	Flags json.RawMessage
}

// GetConfig returns the Orbit config fetched from Fleet server for this instance of OrbitClient.
func (oc *OrbitClient) GetConfig() (*OrbitConfig, error) {
	verb, path := "POST", "/api/fleet/orbit/config"
	var resp orbitGetConfigResponse
	if err := oc.authenticatedRequest(verb, path, &orbitGetConfigRequest{}, &resp); err != nil {
		return nil, err
	}
	return &OrbitConfig{
		Flags: resp.Flags,
	}, nil
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
	params := EnrollOrbitRequest{EnrollSecret: oc.enrollSecret, HardwareUUID: oc.uuid}
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

	orbitNodeKey, err := ioutil.ReadFile(oc.nodeKeyFilePath)
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
				log.Info().Err(err).Msg("enroll failed, retrying")
				return err
			}
		},
		retry.WithInterval(constant.OrbitEnrollRetrySleep),
		retry.WithMaxAttempts(constant.OrbitEnrollMaxRetries),
	); err != nil {
		return "", fmt.Errorf("orbit node key enroll failed, attempts=%d", constant.OrbitEnrollMaxRetries)
	}
	if endpointDoesNotExist {
		return "", errors.New("enroll endpoint does not exist")
	}
	return orbitNodeKey_, nil
}

func (oc *OrbitClient) enrollAndWriteNodeKeyFile() (string, error) {
	orbitNodeKey, err := oc.enroll()
	if err != nil {
		return "", fmt.Errorf("enroll request: %w", err)
	}
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
