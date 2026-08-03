package luks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type LuksDump struct {
	Keyslots map[string]Keyslot `json:"keyslots"` // keyslot -> salt
	Tokens   map[string]Token   `json:"tokens"`
}

type Keyslot struct {
	KDF KDF `json:"kdf"`
}

type KDF struct {
	Salt string `json:"salt"`
}

// Token represents an entry from the LUKS2 "tokens" object. We only need the
// type field to identify TPM2/FIDO2/recovery setups; other fields are ignored.
type Token struct {
	Type string `json:"type"`
}

// EncryptionType values returned by DetectEncryptionType. These are local to
// orbit — the server does not currently persist or care about which one the
// host is on; the value is only used to branch dialog copy when prompting the
// end user during escrow.
const (
	EncryptionTypePassphrase = "passphrase"
	EncryptionTypeTPM2       = "tpm2"
	EncryptionTypeFIDO2      = "fido2"
	EncryptionTypeRecovery   = "recovery"
)

// LUKS2 token-type identifiers emitted by systemd-cryptenroll. These match
// the literal "type" field of entries in the luksDump "tokens" object.
const (
	systemdTPM2Type     = "systemd-tpm2"
	systemdFIDO2Type    = "systemd-fido2"
	systemdRecoveryType = "systemd-recovery"
)

// DetectEncryptionType inspects a LUKS2 dump's tokens and returns the
// best-matching encryption type for the volume. Priority order when multiple
// tokens are present is tpm2 > fido2 > recovery > passphrase. A nil dump, an
// empty tokens map, or unrecognized token types all map to
// EncryptionTypePassphrase.
func DetectEncryptionType(dump *LuksDump) string {
	if dump == nil {
		return EncryptionTypePassphrase
	}

	var hasFIDO2, hasRecovery bool
	for _, tok := range dump.Tokens {
		switch tok.Type {
		case systemdTPM2Type:
			return EncryptionTypeTPM2
		case systemdFIDO2Type:
			hasFIDO2 = true
		case systemdRecoveryType:
			hasRecovery = true
		}
	}

	switch {
	case hasFIDO2:
		return EncryptionTypeFIDO2
	case hasRecovery:
		return EncryptionTypeRecovery
	}
	return EncryptionTypePassphrase
}

// snapdFDETokenSubstr matches LUKS2 token types written by snapd's secboot
// full-disk-encryption stack (e.g. Ubuntu 26 TPM-backed FDE). Unlike
// systemd-cryptenroll — which writes systemd-tpm2 / systemd-recovery tokens —
// snapd owns its own named key slots, so we cannot escrow by adding a key slot
// with cryptsetup and must escrow a snapd recovery key instead.
//
// NOTE: the exact token type string must be confirmed on real Ubuntu 26
// hardware; the substring match is intentionally lenient.
const snapdFDETokenSubstr = "fde"

// FleetRecoveryKeyName is the name of the dedicated snapd recovery key slot
// Fleet creates and escrows. Using a separate, named slot leaves the user's
// install-time "default-recovery" key untouched.
const FleetRecoveryKeyName = "fleet-escrow"

// IsSnapdManaged reports whether the LUKS2 volume is managed by snapd's secboot
// FDE stack (as opposed to a plain passphrase or systemd-cryptenroll volume).
func IsSnapdManaged(dump *LuksDump) bool {
	if dump == nil {
		return false
	}
	for _, tok := range dump.Tokens {
		if strings.Contains(strings.ToLower(tok.Type), snapdFDETokenSubstr) {
			return true
		}
	}
	return false
}

// SnapdFDE abstracts the snapd/secboot tooling used to manage TPM-backed
// full-disk encryption recovery keys. It is an interface so the escrow
// orchestration can be unit tested without snapd present.
//
// There is intentionally no "remove recovery key" operation: neither snapd's
// /v2/system-volumes API nor the snap-tpmctl CLI exposes recovery-key/keyslot
// deletion (only passphrase/PIN auth factors can be removed). A recovery key is
// retired by rotating it (regenerate/replace), which EnsureFleetRecoveryKey
// does, so a failed escrow self-heals on the next attempt.
type SnapdFDE interface {
	// Detect reports whether this host uses snapd-managed TPM-backed FDE.
	Detect(ctx context.Context) (bool, error)
	// EnsureFleetRecoveryKey creates (or regenerates) the Fleet-owned recovery
	// key and returns its plaintext value to escrow.
	EnsureFleetRecoveryKey(ctx context.Context) (string, error)
}

var recoveryKeyRegexp = regexp.MustCompile(`\d{5}(?:-\d{5})+`)

// parseRecoveryKey extracts a snapd recovery key (groups of five digits
// separated by hyphens, e.g. 55055-39320-...) from command output.
func parseRecoveryKey(output string) (string, error) {
	key := recoveryKeyRegexp.FindString(output)
	if key == "" {
		return "", errors.New("no recovery key found in output")
	}
	return key, nil
}

// runRecoveryKeyEscrow escrows a snapd-managed recovery key for hosts using
// TPM-backed full-disk encryption (e.g. Ubuntu 26). Unlike the legacy
// passphrase path it requires no end-user interaction: snapd owns the LUKS key
// slots, so Fleet creates a dedicated recovery key and escrows it silently.
//
// Old Fleet servers reject the recovery-key payload shape (no Salt, no
// KeySlot) with a misleading "passphrase, salt, and key_slot..." error. We
// gate the whole path on CapabilityLUKSRecoveryKeyEscrow so a stale server
// doesn't cause orbit to rotate the fleet-escrow slot's key on every retry
// (see the add-then-replace logic in ensureFleetRecoveryKey).
func (lr *LuksRunner) runRecoveryKeyEscrow(ctx context.Context, snapd SnapdFDE) error {
	if !lr.escrower.GetServerCapabilities().Has(fleet.CapabilityLUKSRecoveryKeyEscrow) {
		log.Warn().Msg("Fleet server does not advertise the LUKS recovery-key escrow capability; skipping this attempt. Upgrade the Fleet server to enable TPM-backed FDE key backup.")
		lr.notifyServerTooOldOnce()
		return nil
	}

	response := LuksResponse{KeyType: fleet.LUKSKeyTypeRecoveryKey}

	log.Info().Msg("creating and enrolling snapd-managed FDE recovery key for escrow")
	recoveryKey, err := snapd.EnsureFleetRecoveryKey(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to create snapd-managed recovery key; reporting escrow error to Fleet")
		response.Err = fmt.Sprintf("creating Fleet recovery key: %s", err)
		lr.recordRecoveryKeyFailure()
		if sendErr := lr.escrower.SendLinuxKeyEscrowResponse(response); sendErr != nil {
			return fmt.Errorf("reporting recovery key escrow error: %w", sendErr)
		}
		return fmt.Errorf("creating Fleet recovery key: %w", err)
	}

	response.Passphrase = recoveryKey
	log.Debug().Msg("sending escrowed recovery key to the Fleet server")
	if err := lr.escrower.SendLinuxKeyEscrowResponse(response); err != nil {
		// The server did not record the key. snapd exposes no way to delete a
		// recovery-key slot, so we cannot roll the enrolled key back — but it is
		// harmless (its secret was never stored anywhere) and the host stays
		// pending escrow, so the next attempt regenerates and replaces it in
		// place. Escrow therefore self-heals on retry.
		log.Error().Err(err).Msg("failed to escrow recovery key to Fleet; the host stays pending and will retry on the next check-in")
		lr.recordRecoveryKeyFailure()
		return fmt.Errorf("escrowing recovery key: %w", err)
	}

	log.Info().Msg("snapd-managed FDE recovery key escrowed to Fleet")
	lr.recordRecoveryKeySuccess()
	return nil
}

// recordRecoveryKeyFailure counts a consecutive failure and, once the
// threshold is crossed, shows a one-shot notification informing the user that
// their disk recovery key could not be escrowed. The one-shot flag stays set
// until a subsequent success clears it, so we never spam.
func (lr *LuksRunner) recordRecoveryKeyFailure() {
	lr.mu.Lock()
	lr.recoveryKeyFailures++
	shouldNotify := lr.recoveryKeyFailures >= recoveryKeyFailureNotifyThreshold && !lr.recoveryKeyNotified
	if shouldNotify {
		lr.recoveryKeyNotified = true
	}
	lr.mu.Unlock()

	// Dialog call outside the lock: ShowInfo can block for up to a minute
	// while the user reads the message and we don't want to serialize other
	// receivers behind it.
	if shouldNotify {
		lr.showInfo(recoveryKeyEscrowFailedTitle, recoveryKeyEscrowFailedText)
	}
}

// recordRecoveryKeySuccess resets the failure streak so a later outage
// triggers the notification again.
func (lr *LuksRunner) recordRecoveryKeySuccess() {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.recoveryKeyFailures = 0
	lr.recoveryKeyNotified = false
	lr.serverTooOldNotified = false
}

// notifyServerTooOldOnce shows a one-shot notification when the Fleet server
// is too old to accept recovery-key escrow. This is distinct from the retry
// notification because the situation is deterministic — no amount of retrying
// will make it work until the admin upgrades — so we notify immediately, not
// after N ticks.
func (lr *LuksRunner) notifyServerTooOldOnce() {
	lr.mu.Lock()
	shouldNotify := !lr.serverTooOldNotified
	if shouldNotify {
		lr.serverTooOldNotified = true
	}
	lr.mu.Unlock()

	if shouldNotify {
		lr.showInfo(recoveryKeyServerTooOldTitle, recoveryKeyServerTooOldText)
	}
}

// showInfo shows an info dialog if a notifier is available. On headless hosts
// (no zenity/kdialog installed, or no logged-in GUI user) this is a no-op
// beyond a log line; the failure surfaces via the server-side "pending escrow"
// state on the host's page in the Fleet UI.
func (lr *LuksRunner) showInfo(title, text string) {
	if lr.notifier == nil {
		log.Debug().Str("title", title).Msg("no dialog notifier available; skipping user-facing recovery-key notification")
		return
	}
	if err := lr.notifier.ShowInfo(dialog.InfoOptions{
		Title:   title,
		Text:    text,
		TimeOut: 1 * time.Minute,
	}); err != nil {
		log.Info().Err(err).Str("title", title).Msg("failed to show recovery-key escrow notification")
	}
}

type KeyEscrower interface {
	SendLinuxKeyEscrowResponse(LuksResponse) error
	// GetServerCapabilities returns the capabilities the Fleet server most
	// recently advertised via the X-Fleet-Capabilities header. Used to gate
	// the snapd/TPM-backed FDE recovery-key escrow path on servers that
	// understand the recovery-key payload shape.
	GetServerCapabilities() fleet.CapabilityMap
}

// recoveryKeyFailureNotifyThreshold is the number of consecutive
// runRecoveryKeyEscrow failures at which the user is shown a one-shot
// notification. Config-receiver ticks run roughly every 30s, so 5 attempts is
// ~2.5 minutes of silent retries before we bother the user.
const recoveryKeyFailureNotifyThreshold = 5

// Copy for the snapd/TPM-backed FDE recovery-key path. Unlike the passphrase
// flow these fire only on persistent failure — successful escrow stays silent
// because the whole point of the TPM path is minimal end-user friction.
const (
	recoveryKeyEscrowFailedTitle = "Disk encryption"
	recoveryKeyEscrowFailedText  = "Fleet couldn't back up your disk recovery key. Please contact your IT admin."
	recoveryKeyServerTooOldTitle = "Disk encryption"
	recoveryKeyServerTooOldText  = "Your Fleet server needs an update before it can back up your disk recovery key. Please contact your IT admin."
)

type LuksRunner struct {
	escrower KeyEscrower
	notifier dialog.Dialog

	// State for the snapd/TPM-backed FDE recovery-key path. Protected by mu
	// because Run() may be invoked from the config-receiver loop and we want
	// to be safe against future concurrent tick semantics. All fields reset
	// on a successful escrow.
	mu                   sync.Mutex
	recoveryKeyFailures  int
	recoveryKeyNotified  bool // one-shot for the escrow-failed notification
	serverTooOldNotified bool // one-shot for the capability-missing notification
}

type LuksResponse struct {
	// Passphrase is a newly created passphrase generated by fleetd for securing the LUKS volume.
	// This passphrase will be securely escrowed to the server.
	Passphrase string

	// KeySlot specifies the LUKS key slot where this new passphrase was created.
	// It is currently not used, but may be useful in the future for passphrase rotation.
	KeySlot *uint

	// Salt is the salt used to generate the LUKS key.
	Salt string

	// KeyType identifies how the escrowed secret unlocks the volume. Empty
	// means the legacy passphrase path (with Salt + KeySlot);
	// fleet.LUKSKeyTypeRecoveryKey means a snapd-managed TPM-backed FDE recovery
	// key, which has no Salt or KeySlot.
	KeyType string

	// Err is the error message that occurred during the escrow process.
	Err string
}

func New(escrower KeyEscrower) *LuksRunner {
	return &LuksRunner{
		escrower: escrower,
	}
}

func extractJSON(input []byte) ([]byte, error) {
	// Regular expression to extract JSON
	re := regexp.MustCompile(`(?s)\{.*\}`)
	match := re.FindString(string(input))
	if match == "" {
		return nil, errors.New("no JSON found")
	}
	return []byte(match), nil
}
