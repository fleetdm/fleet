// Package managedaccount creates and maintains the Fleet-managed local admin account on Windows
// hosts, and escrows its password to the Fleet server.
//
// The server asks for the account by setting the CreateWindowsManagedLocalAccount notification on
// the orbit config response, and stops asking once this host escrows a password for its current MDM
// enrollment. That makes the server the owner of "is this done", so there is deliberately no local
// terminal state here: whenever Fleet asks, fleetd provisions the account and escrows the password.
// Every step is idempotent, so being asked again is always safe. Keeping a local "already done"
// marker would be able to disagree with the server and silently stall the flow, for instance after a
// re-enrollment that did not wipe the disk.
package managedaccount

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// passwordLength is the length of the generated password. The server accepts up to 256 characters;
// 32 random characters across four classes is far beyond what a local account needs while staying
// comfortably inside any password-length policy.
const passwordLength = 32

// Character classes for the generated password. Windows' default complexity policy requires three of
// four, so all four are guaranteed. The symbol set deliberately omits quotes, backslashes and
// spaces, which are awkward to type at the "Other user" sign-in prompt during a break-glass login.
const (
	lowerChars  = "abcdefghijkmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars  = "23456789"
	symbolChars = "!#%&*+-=?@^_"
)

// Escrower sends the managed local account password to the Fleet server. Satisfied by
// *service.OrbitClient; an interface so tests do not need a server.
type Escrower interface {
	SendManagedLocalAccountPassword(password, clientError string) error
}

// provisionFunc creates or updates the managed local admin account and hides it from the sign-in
// screen. Implemented per platform; a no-op everywhere except Windows.
type provisionFunc func(username, password string) error

// Receiver reacts to the CreateWindowsManagedLocalAccount notification.
type Receiver struct {
	escrower Escrower

	// provision is indirected so tests can exercise the flow without touching Windows APIs. nil means
	// use the platform implementation.
	provision provisionFunc

	// mu keeps a single provisioning attempt in flight. Held for the duration of the background
	// goroutine, so a notification that arrives again while work is running is dropped rather than
	// starting a second account reset.
	mu sync.Mutex

	// done, when non-nil, is closed after each completed attempt. Tests only.
	done chan struct{}
}

// New returns a Receiver that escrows through the given Escrower.
func New(escrower Escrower) *Receiver {
	return &Receiver{escrower: escrower}
}

// Run implements fleet.OrbitConfigReceiver. It returns immediately; provisioning happens in the
// background so a slow Windows API call or HTTP request never gates the config receiver loop.
func (r *Receiver) Run(cfg *fleet.OrbitConfig) error {
	if cfg == nil || !cfg.Notifications.CreateWindowsManagedLocalAccount {
		return nil
	}
	r.attempt()
	return nil
}

func (r *Receiver) attempt() {
	// TryLock rather than Lock: if an attempt is already running, drop this one instead of queueing a
	// second account reset behind it. The server keeps asking until an escrow succeeds, so nothing is
	// lost by skipping.
	if !r.mu.TryLock() {
		log.Debug().Msg("managed local account: provisioning already in progress, skipping")
		return
	}
	go func() {
		defer r.mu.Unlock()
		r.createAndEscrow()
		if r.done != nil {
			close(r.done)
		}
	}()
}

// createAndEscrow generates a password, provisions the account, and escrows the password. Any
// failure before the escrow returns without recording success, so the next config fetch retries the
// whole flow; the provisioning step resets the password of an existing account, which is what makes
// that retry safe.
func (r *Receiver) createAndEscrow() {
	password, err := generatePassword()
	if err != nil {
		// Nothing to report to the server: we never touched the device, and the next poll retries.
		log.Error().Err(err).Msg("managed local account: generating password")
		return
	}

	provision := r.provision
	if provision == nil {
		provision = provisionAccount
	}

	if err := provision(fleet.ManagedLocalAccountUsername, password); err != nil {
		log.Error().Err(err).Msg("managed local account: creating account")
		// Tell the server why, so it surfaces on the host instead of only in this log. The server
		// records the failure and keeps asking, so this is a report, not a terminal state.
		if escrowErr := r.escrower.SendManagedLocalAccountPassword("", err.Error()); escrowErr != nil {
			log.Error().Err(escrowErr).Msg("managed local account: reporting creation failure")
		}
		return
	}

	if err := r.escrower.SendManagedLocalAccountPassword(password, ""); err != nil {
		// The account now exists with a password Fleet does not know. That is recovered by the next
		// notification: provisioning resets the password and escrows the new one.
		log.Error().Err(err).Msg("managed local account: escrowing password")
		return
	}

	log.Info().Msg("managed local account: created and escrowed")
}

// generatePassword returns a cryptographically random password of passwordLength characters that
// contains at least one character from each class.
func generatePassword() (string, error) {
	classes := []string{lowerChars, upperChars, digitChars, symbolChars}
	all := strings.Join(classes, "")

	chars := make([]byte, 0, passwordLength)
	// Seed one character per class so complexity requirements are met regardless of how the remainder
	// falls out, then fill the rest from the full alphabet.
	for _, class := range classes {
		c, err := randomChar(class)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}
	for len(chars) < passwordLength {
		c, err := randomChar(all)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}

	// Shuffle so the seeded characters are not always in the first four positions.
	for i := len(chars) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", fmt.Errorf("shuffling password: %w", err)
		}
		chars[i], chars[j.Int64()] = chars[j.Int64()], chars[i]
	}
	return string(chars), nil
}

func randomChar(alphabet string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
	if err != nil {
		return 0, fmt.Errorf("reading random number: %w", err)
	}
	return alphabet[n.Int64()], nil
}
