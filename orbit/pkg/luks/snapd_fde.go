//go:build linux

package luks

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/rs/zerolog/log"
)

// systemVolumesPath is the snapd REST API endpoint for managing FDE key slots
// (recovery keys, passphrases) on system volumes at runtime. snapd 2.74
// (shipping in Ubuntu 26.04) added post-install recovery-key management here,
// including support for management agents enrolling a dedicated, named recovery
// key. The endpoint, the actions below, and the request/response field names
// are confirmed present in the released snapd tags 2.74 and 2.75 (current stable
// line is 2.75.x/2.76), not just master
// (daemon/api_system_volumes.go, overlord/fdestate/fdestate.go).
const systemVolumesPath = "/v2/system-volumes"

// snapdMinVersion is the earliest snapd version that supports the runtime
// recovery-key management actions on /v2/system-volumes (generate/add/replace/
// check-recovery-key). Older snapd builds have `ubuntu-fde` LUKS2 tokens on
// disk — so IsSnapdManaged still returns true — but reject the actions with
// "this action is not supported on this system". Preflighting the version
// lets us surface a clear "your snapd is too old" error to the admin instead
// of the raw snapd 400.
const snapdMinVersion = "2.74.0"

// snapd /v2/system-volumes action values.
const (
	actionGenerateRecoveryKey = "generate-recovery-key"
	actionAddRecoveryKey      = "add-recovery-key"
	actionReplaceRecoveryKey  = "replace-recovery-key"
	actionCheckRecoveryKey    = "check-recovery-key"
)

// keyslotRef mirrors snapd's fdestate.KeyslotRef. A name-only entry (empty
// container-role) implicitly targets all system containers (system-data and
// system-save) as of snapd 2.74.
type keyslotRef struct {
	Name          string `json:"name"`
	ContainerRole string `json:"container-role,omitempty"`
}

type generateRecoveryKeyRequest struct {
	Action string `json:"action"`
}

type generateRecoveryKeyResponse struct {
	RecoveryKey string `json:"recovery-key"`
	KeyID       string `json:"key-id"`
}

// recoveryKeyActionRequest is the request body for the add/replace/check
// recovery-key actions. Fields are omitted when empty so the same struct serves
// all three.
type recoveryKeyActionRequest struct {
	Action      string       `json:"action"`
	KeyID       string       `json:"key-id,omitempty"`
	Keyslots    []keyslotRef `json:"keyslots,omitempty"`
	RecoveryKey string       `json:"recovery-key,omitempty"`
}

// snapdSocketFDE manages FDE recovery keys via the snapd REST API socket.
type snapdSocketFDE struct {
	client *snapdClient
}

func newSnapdSocketFDE() *snapdSocketFDE {
	return &snapdSocketFDE{client: newSnapdClient()}
}

// ensureFleetRecoveryKey generates a recovery key and enrolls it under the
// dedicated Fleet-owned key slot name, returning the recovery key to escrow.
func (s *snapdSocketFDE) ensureFleetRecoveryKey(ctx context.Context) (string, error) {
	// Step 0: preflight snapd's version. The /v2/system-volumes actions we
	// use (generate/add/check-recovery-key) landed in snapd 2.74. Older snapd
	// still has `ubuntu-fde` LUKS2 tokens on disk (so IsSnapdManaged returns
	// true) but rejects the action with a 400 "not supported on this system".
	// Fail fast with a version-specific message so the operator sees what
	// actually needs to change.
	if err := s.checkSnapdVersion(ctx); err != nil {
		return "", err
	}

	// Step 1: generate a recovery key. snapd answers synchronously with the key
	// value and a transient id used to enroll it.
	log.Debug().Str("action", actionGenerateRecoveryKey).Msg("requesting snapd to generate a recovery key")
	var gen generateRecoveryKeyResponse
	if err := s.client.requestSync(ctx, http.MethodPost, systemVolumesPath,
		generateRecoveryKeyRequest{Action: actionGenerateRecoveryKey}, &gen); err != nil {
		// snapd rejects the action with "not supported on this system" when
		// it sees `ubuntu-fde` tokens but does not consider itself the
		// authoritative FDE manager (e.g. `snap-tpmctl status` reports the
		// FDE system as indeterminate). Version-upgrading snapd does not fix
		// this; the operator has to resolve the FDE state on the host, so
		// surface a specific message instead of the raw snapd 400.
		var apiErr *snapdAPIError
		if errors.As(err, &apiErr) && apiErr.isUnsupportedOperation() {
			log.Warn().Err(err).Msg("snapd refuses to manage recovery keys on this host; the FDE system is likely not fully snapd-managed (check `snap-tpmctl status`)")
			return "", fmt.Errorf("snapd does not consider this host a snapd-managed FDE system, so recovery-key escrow is unavailable. Run `snap-tpmctl status` on the host — if it reports \"the fde system is indeterminate\", the LUKS2 volume is not owned by snapd's secboot stack and orbit cannot escrow a recovery key here. Underlying snapd error: %s", apiErr.Message)
		}
		return "", fmt.Errorf("generating snapd recovery key: %w", err)
	}
	if gen.RecoveryKey == "" || gen.KeyID == "" {
		return "", errors.New("snapd returned an incomplete recovery key")
	}
	// Log the transient key id and the key length, never the key itself.
	log.Debug().Str("key_id", gen.KeyID).Int("recovery_key_len", len(gen.RecoveryKey)).
		Msg("snapd generated a recovery key")

	// Step 2: enroll the generated key under the Fleet-owned name (a name-only
	// keyslot targets both system containers). This is asynchronous because it
	// mutates the LUKS volume. On first run the slot does not exist so
	// add-recovery-key applies; on a retry the slot already exists, so we fall
	// back to replace-recovery-key to rotate the secret in place. The fallback
	// is gated on the error looking like a "resource already exists" conflict
	// so unrelated failures (auth, transport, malformed request) surface as
	// errors instead of silently rotating an existing key we couldn't add for
	// a different reason.
	slots := []keyslotRef{{Name: FleetRecoveryKeyName}}
	log.Debug().Str("action", actionAddRecoveryKey).Str("keyslot", FleetRecoveryKeyName).Str("key_id", gen.KeyID).
		Msg("enrolling recovery key under Fleet keyslot")
	addErr := s.client.requestAsync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
		Action: actionAddRecoveryKey, KeyID: gen.KeyID, Keyslots: slots,
	}, nil)
	if addErr != nil {
		var apiErr *snapdAPIError
		if !errors.As(addErr, &apiErr) || !apiErr.isConflict() {
			return "", fmt.Errorf("enrolling snapd recovery key: %w", addErr)
		}
		log.Debug().Err(addErr).Str("action", actionReplaceRecoveryKey).Str("keyslot", FleetRecoveryKeyName).
			Msg("add-recovery-key reports the slot already exists; retrying with replace-recovery-key")
		if rerr := s.client.requestAsync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
			Action: actionReplaceRecoveryKey, KeyID: gen.KeyID, Keyslots: slots,
		}, nil); rerr != nil {
			return "", fmt.Errorf("enrolling snapd recovery key (add: %v; replace: %w)", addErr, rerr)
		}
	}

	// Step 3: validate the freshly enrolled key before escrowing it, mirroring
	// the passphrase flow's post-add validation. check-recovery-key is
	// synchronous and returns success with a null result.
	log.Debug().Str("action", actionCheckRecoveryKey).Msg("validating enrolled recovery key")
	if err := s.client.requestSync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
		Action: actionCheckRecoveryKey, RecoveryKey: gen.RecoveryKey,
	}, nil); err != nil {
		return "", fmt.Errorf("validating snapd recovery key: %w", err)
	}

	log.Info().Str("keyslot", FleetRecoveryKeyName).Msg("snapd-managed recovery key enrolled and validated")
	return gen.RecoveryKey, nil
}

// checkSnapdVersion queries snapd's system-info and returns an error whose
// message explicitly names the minimum required version if the running snapd
// is older than snapdMinVersion. snapd's version string may carry a
// distribution suffix (e.g. "2.68.5+24.04") which the Masterminds/semver
// parser handles as a build/pre-release tag; comparison is on the numeric
// major.minor.patch part.
func (s *snapdSocketFDE) checkSnapdVersion(ctx context.Context) error {
	info, err := s.client.systemInfo(ctx)
	if err != nil {
		return fmt.Errorf("querying snapd system info: %w", err)
	}
	if info.Version == "" {
		return errors.New("snapd system-info did not return a version")
	}

	got, err := semver.NewVersion(info.Version)
	if err != nil {
		return fmt.Errorf("parsing snapd version %q: %w", info.Version, err)
	}
	minVer := semver.MustParse(snapdMinVersion)
	if got.LessThan(minVer) {
		log.Warn().Str("snapd_version", info.Version).Str("min_version", snapdMinVersion).
			Msg("snapd is too old for TPM-backed FDE recovery-key management")
		return fmt.Errorf("snapd %s does not support TPM-backed FDE recovery-key management; upgrade snapd to %s or later (Ubuntu 26.04+)", info.Version, snapdMinVersion)
	}
	log.Debug().Str("snapd_version", info.Version).Msg("snapd version supports recovery-key management")
	return nil
}
