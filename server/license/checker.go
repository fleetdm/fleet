package license

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/go-kit/kit/log"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/version"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

const (
	defaultPollFrequency     = 8 * time.Hour
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
		chk.pollFrequency = freq
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
		logger:        log.NewNopLogger(),
		ds:            ds,
		pollFrequency: defaultPollFrequency,
		client:        &http.Client{Timeout: defaultHttpClientTimeout},
		url:           licenseEndpointURL,
		ticker:        defaultTicker,
		finish:        make(chan struct{}),
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
		chk.logger.Log("poll-frequency", cc.pollFrequency, "http-timeout", cc.client.Timeout)

		for {
			select {
			case <-chk.finish:
				chk.logger.Log("msg", "finishing")
				return
			case <-chk.ticker.Chan():
				chk.RunLicenseCheck(context.Background())
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

// addVersionInfo parses the license URL and adds the current revision of the
// kolide binary to the query params. The reported revision is set using
// ldflags by the make command, otherwise defaults to 'unknown'.
func addVersionInfo(licenseURL string) (*url.URL, error) {
	ur, err := url.Parse(licenseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "license checker failed to parse URL string %q", licenseURL)
	}
	revision := version.Version().Revision
	q := ur.Query()
	q.Set("version", revision)
	ur.RawQuery = q.Encode()
	return ur, nil
}

func (cc *Checker) RunLicenseCheck(ctx context.Context) {
	cc.logger.Log("msg", "begin license check")
	defer cc.logger.Log("msg", "ending license check")

	license, err := cc.ds.License()
	if err != nil {
		cc.logger.Log("msg", "couldn't fetch license", "err", err)
		return
	}
	claims, err := license.Claims()
	if err != nil {
		cc.logger.Log("msg", "fetching claims", "err", err)
		return
	}

	licenseURL, err := addVersionInfo(fmt.Sprintf("%s/%s", cc.url, claims.LicenseUUID))
	if err != nil {
		cc.logger.Log("msg", "adding version information to license", "err", err)
		return
	}

	resp, err := cc.client.Get(licenseURL.String())
	if err != nil {
		cc.logger.Log("msg", fmt.Sprintf("fetching %s", licenseURL.String()), "err", err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var revInfo revokeInfo
		err = json.NewDecoder(resp.Body).Decode(&revInfo)
		if err != nil {
			cc.logger.Log("msg", "decoding response", "err", err)
			return
		}
		err = cc.ds.RevokeLicense(revInfo.Revoked)
		if err != nil {
			cc.logger.Log("msg", "revoke status", "err", err)
			return
		}
		// success
		cc.logger.Log("msg", fmt.Sprintf("license revocation status retrieved succesfully, revoked: %t", revInfo.Revoked))
	case http.StatusNotFound:
		var revInfo revokeError
		err = json.NewDecoder(resp.Body).Decode(&revInfo)
		if err != nil {
			cc.logger.Log("msg", "decoding response", "err", err)
			return
		}
		cc.logger.Log("msg", "host response", "err", fmt.Sprintf("status: %d error: %s", revInfo.Status, revInfo.Error))
	default:
		cc.logger.Log("msg", "host response", "err", fmt.Sprintf("unexpected response status from host, status %s", resp.Status))
	}
}
