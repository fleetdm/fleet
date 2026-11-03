package entratoken

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const (
	// refreshInterval re-reports the token so device ID and user changes reach Fleet.
	refreshInterval = time.Hour
	retryInterval   = 10 * time.Minute
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

	mu      sync.Mutex
	running bool
	lastReq fleet.OrbitEntraDeviceTokenRequest
	nextRun time.Time
}

// NewRunner returns a Runner for the host with the given hardware UUID.
func NewRunner(client tokenSender, hardwareUUID string) *Runner {
	return &Runner{
		client:       client,
		hardwareUUID: hardwareUUID,
		getToken:     GetIDTokenAsLoggedInUser,
	}
}

// Run starts a report in the background when the request changed or is due.
func (r *Runner) Run(cfg *fleet.OrbitConfig) error {
	req := cfg.EntraDeviceTokenRequest
	if req == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running || (*req == r.lastReq && time.Now().Before(r.nextRun)) {
		return nil
	}
	r.running = true
	r.lastReq = *req
	go r.report(*req)
	return nil
}

func (r *Runner) report(req fleet.OrbitEntraDeviceTokenRequest) {
	err := r.reportOnce(req)
	next := refreshInterval
	if err != nil {
		next = retryInterval
		if errors.Is(err, ErrNoUserLoggedIn) {
			log.Debug().Msg("entra device token: no user logged in")
		} else {
			log.Info().Err(err).Msg("entra device token: report failed")
		}
	}
	r.mu.Lock()
	r.running = false
	r.nextRun = time.Now().Add(next)
	r.mu.Unlock()
}

func (r *Runner) reportOnce(req fleet.OrbitEntraDeviceTokenRequest) error {
	nonceHost, _, _ := strings.Cut(req.Nonce, ".")
	if r.hardwareUUID == "" || !strings.EqualFold(nonceHost, r.hardwareUUID) {
		return errors.New("nonce was not generated for this host")
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
