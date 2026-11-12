package entratoken

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const (
	// refreshInterval re-reports the token so device ID and user changes reach Fleet.
	refreshInterval = time.Hour
	reportTimeout   = time.Minute
)

// ErrNoUserLoggedIn is returned when no user is logged in to request the token for.
var ErrNoUserLoggedIn = errors.New("no user logged in")

type tokenSender interface {
	SendEntraDeviceToken(idToken string) error
}

// Runner is an orbit config receiver that reports the Entra device token Fleet requests.
type Runner struct {
	client       tokenSender
	hardwareUUID string
	getToken     func(ctx context.Context, r Request) (string, error)
	// Failed reports are retried on a timer with exponential backoff instead of waiting for the next
	// orbit config, so a host is verified soon after its user registers with Platform SSO.
	minRetry time.Duration
	maxRetry time.Duration

	mu         sync.Mutex
	running    bool
	lastReq    fleet.OrbitEntraDeviceTokenRequest
	nextRun    time.Time
	failures   int
	retryTimer *time.Timer
}

// NewRunner returns a Runner for the host with the given hardware UUID.
func NewRunner(client tokenSender, hardwareUUID string) *Runner {
	return &Runner{
		client:       client,
		hardwareUUID: hardwareUUID,
		getToken:     GetIDTokenAsLoggedInUser,
		minRetry:     time.Minute,
		maxRetry:     10 * time.Minute,
	}
}

// Run starts a report in the background when the request changed or is due.
func (r *Runner) Run(cfg *fleet.OrbitConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req := cfg.EntraDeviceTokenRequest
	if req == nil {
		r.lastReq = fleet.OrbitEntraDeviceTokenRequest{}
		r.stopRetryLocked()
		return nil
	}
	if *req != r.lastReq {
		r.failures = 0
		r.stopRetryLocked()
	} else if r.running || time.Now().Before(r.nextRun) {
		return nil
	}
	if r.running {
		// The running report starts the new request when it finishes.
		r.lastReq = *req
		return nil
	}
	r.startLocked(*req)
	return nil
}

func (r *Runner) startLocked(req fleet.OrbitEntraDeviceTokenRequest) {
	r.running = true
	r.lastReq = req
	go r.report(req)
}

func (r *Runner) stopRetryLocked() {
	if r.retryTimer != nil {
		r.retryTimer.Stop()
		r.retryTimer = nil
	}
}

func (r *Runner) report(req fleet.OrbitEntraDeviceTokenRequest) {
	err := r.reportOnce(req)
	switch {
	case err == nil:
		log.Debug().Msg("entra device token: reported")
	case errors.Is(err, ErrNoUserLoggedIn):
		log.Debug().Msg("entra device token: no user logged in")
	default:
		log.Info().Err(err).Msg("entra device token: report failed")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
	if r.lastReq == (fleet.OrbitEntraDeviceTokenRequest{}) {
		// Fleet stopped requesting the token while reporting.
		return
	}
	if req != r.lastReq {
		// The request changed while reporting; report the new one now.
		r.failures = 0
		r.startLocked(r.lastReq)
		return
	}
	if err == nil {
		r.failures = 0
		r.nextRun = time.Now().Add(refreshInterval)
		return
	}
	delay := r.maxRetry
	if r.failures < 10 {
		delay = min(r.minRetry<<r.failures, r.maxRetry)
	}
	r.failures++
	r.nextRun = time.Now().Add(delay)
	r.stopRetryLocked()
	r.retryTimer = time.AfterFunc(delay, func() { r.retry(req) })
}

func (r *Runner) retry(req fleet.OrbitEntraDeviceTokenRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running || req != r.lastReq {
		return
	}
	r.startLocked(req)
}

func (r *Runner) reportOnce(req fleet.OrbitEntraDeviceTokenRequest) error {
	if !nonceIsForHost(req.Nonce, r.hardwareUUID) {
		return errNonceForAnotherHost
	}
	ctx, cancel := context.WithTimeout(context.Background(), reportTimeout)
	defer cancel()
	token, err := r.getToken(ctx, Request{
		TenantID:    req.TenantID,
		ClientID:    req.ClientID,
		RedirectURI: DefaultRedirectURI,
		Nonce:       req.Nonce,
	})
	if err != nil {
		return err
	}
	return r.client.SendEntraDeviceToken(token)
}
