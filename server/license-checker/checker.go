package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/go-kit/kit/log"
	"github.com/kolide/kolide/server/kolide"
)

const (
	defaultPollFrequency     = time.Hour
	defaultHttpClientTimeout = 10 * time.Second
)

type timer struct {
	*time.Ticker
}

func (t *timer) Chan() <-chan time.Time {
	return t.C
}

type revokeInfo struct {
	UUID    string `json:"uuid"`
	Revoked bool   `json:"revoked"`
}

type revokeError struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// Checker checks remote kolide/cloud app for license revocation
// status
type Checker struct {
	ds            kolide.Datastore
	logger        log.Logger
	url           string
	pollFrequency time.Duration
	ticker        clock.Ticker
	client        *http.Client
	finish        chan struct{}
}

type Option func(chk *Checker)

// Logger set the logger that will be used by the Checker
func Logger(logger log.Logger) Option {
	return func(chk *Checker) {
		chk.logger = logger
	}
}

// HTTPClient supply your own http client
func HTTPClient(client *http.Client) Option {
	return func(chk *Checker) {
		chk.client = client
	}
}

func PollFrequency(freq time.Duration) Option {
	ticker := &timer{
		Ticker: time.NewTicker(freq),
	}
	return func(chk *Checker) {
		chk.ticker = ticker
	}
}

// NewChecker instantiates a service that will check periodically to see if a license
// is revoked.  licenseEndpointURL is the root url for kolide/cloud server.  For example
// https://cloud.kolide.co/api/v0/licenses
// You may optionally set a logger, and/or supply a polling frequency that defines
// how often we check for revocation.
func NewChecker(ds kolide.Datastore, licenseEndpointURL string, opts ...Option) *Checker {
	defaultTicker := &timer{
		Ticker: time.NewTicker(defaultPollFrequency),
	}
	response := &Checker{
		logger: log.NewNopLogger(),
		ds:     ds,
		client: &http.Client{Timeout: defaultHttpClientTimeout},
		url:    licenseEndpointURL,
		ticker: defaultTicker,
		finish: make(chan struct{}),
	}
	for _, o := range opts {
		o(response)
	}

	response.logger = log.NewContext(response.logger).With("component", "license-checker")
	return response
}

var wait sync.WaitGroup

// Start begins checking for license revocation. Note that start can only
// be called once. If Stop is called you must create a new checker to use
// it again.
func (cc *Checker) Start() error {
	if cc.finish == nil {
		return errors.New("start called on stopped checker")
	}
	// pass in copy of receiver to avoid race conditions
	go func(chk Checker, wait *sync.WaitGroup) {
		wait.Add(1)
		defer wait.Done()
		chk.logger.Log("msg", "starting")
		for {
			select {
			case <-chk.finish:
				chk.logger.Log("msg", "finishing")
				return
			case <-chk.ticker.Chan():
				updateLicenseRevocation(&chk)
			}
		}
	}(*cc, &wait)

	return nil
}

// Stop ends checking for license revocation.
func (cc *Checker) Stop() {
	cc.ticker.Stop()
	close(cc.finish)
	wait.Wait()
	cc.finish = nil
}

func updateLicenseRevocation(chk *Checker) {
	chk.logger.Log("msg", "begin license check")
	defer chk.logger.Log("msg", "ending license check")

	license, err := chk.ds.License()
	if err != nil {
		chk.logger.Log("msg", "couldn't fetch license", "err", err)
		return
	}
	claims, err := license.Claims()
	if err != nil {
		chk.logger.Log("msg", "fetching claims", "err", err)
		return
	}
	url := fmt.Sprintf("%s/%s", chk.url, claims.LicenseUUID)
	resp, err := chk.client.Get(url)
	if err != nil {
		chk.logger.Log("msg", fmt.Sprintf("fetching %s", url), "err", err)
		return
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		var revInfo revokeInfo
		err = json.NewDecoder(resp.Body).Decode(&revInfo)
		if err != nil {
			chk.logger.Log("msg", "decoding response", "err", err)
			return
		}
		err = chk.ds.RevokeLicense(revInfo.Revoked)
		if err != nil {
			chk.logger.Log("msg", "revoke status", "err", err)
			return
		}
		// success
		chk.logger.Log("msg", fmt.Sprintf("license revocation status retrieved succesfully, revoked: %t", revInfo.Revoked))
	case http.StatusNotFound:
		var revInfo revokeError
		err = json.NewDecoder(resp.Body).Decode(&revInfo)
		if err != nil {
			chk.logger.Log("msg", "decoding response", "err", err)
			return
		}
		chk.logger.Log("msg", "host response", "err", fmt.Sprintf("status: %d error: %s", revInfo.Status, revInfo.Error))
	default:
		chk.logger.Log("msg", "host response", "err", fmt.Sprintf("unexpected response status from host, status %s", resp.Status))
	}
}
