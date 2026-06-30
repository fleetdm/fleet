package luks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// systemVolumesPath is the snapd REST API endpoint for managing FDE key slots
// (recovery keys, passphrases) on system volumes at runtime. snapd 2.74
// (shipping in Ubuntu 26.04) added post-install recovery-key management here,
// including support for management agents enrolling a dedicated, named recovery
// key. Contract confirmed against canonical/snapd master
// (daemon/api_system_volumes.go, overlord/fdestate/fdestate.go).
const systemVolumesPath = "/v2/system-volumes"

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
	// Step 1: generate a recovery key. snapd answers synchronously with the key
	// value and a transient id used to enroll it.
	var gen generateRecoveryKeyResponse
	if err := s.client.requestSync(ctx, http.MethodPost, systemVolumesPath,
		generateRecoveryKeyRequest{Action: actionGenerateRecoveryKey}, &gen); err != nil {
		return "", fmt.Errorf("generating snapd recovery key: %w", err)
	}
	if gen.RecoveryKey == "" || gen.KeyID == "" {
		return "", errors.New("snapd returned an incomplete recovery key")
	}

	// Step 2: enroll the generated key under the Fleet-owned name (a name-only
	// keyslot targets both system containers). This is asynchronous because it
	// mutates the LUKS volume. On first run the slot does not exist so
	// add-recovery-key applies; on a retry the slot already exists, so fall back
	// to replace-recovery-key to rotate the secret in place.
	slots := []keyslotRef{{Name: FleetRecoveryKeyName}}
	if err := s.client.requestAsync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
		Action: actionAddRecoveryKey, KeyID: gen.KeyID, Keyslots: slots,
	}, nil); err != nil {
		if rerr := s.client.requestAsync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
			Action: actionReplaceRecoveryKey, KeyID: gen.KeyID, Keyslots: slots,
		}, nil); rerr != nil {
			return "", fmt.Errorf("enrolling snapd recovery key (add: %v; replace: %w)", err, rerr)
		}
	}

	// Step 3: validate the freshly enrolled key before escrowing it, mirroring
	// the passphrase flow's post-add validation. check-recovery-key is
	// synchronous and returns success with a null result.
	if err := s.client.requestSync(ctx, http.MethodPost, systemVolumesPath, recoveryKeyActionRequest{
		Action: actionCheckRecoveryKey, RecoveryKey: gen.RecoveryKey,
	}, nil); err != nil {
		return "", fmt.Errorf("validating snapd recovery key: %w", err)
	}

	return gen.RecoveryKey, nil
}
