//go:build linux

package luks

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedEntry is a single canned response from the fake dialog's ShowEntry.
type scriptedEntry struct {
	value []byte
	err   error
}

// fakeDialog is a dialog.Dialog test double. ShowEntry returns the scripted
// entries in order and records the text it was shown with each call.
type fakeDialog struct {
	entries   []scriptedEntry
	callIdx   int
	shownText []string
	infoTexts []string
}

func (f *fakeDialog) ShowEntry(opts dialog.EntryOptions) ([]byte, error) {
	f.shownText = append(f.shownText, opts.Text)
	if f.callIdx >= len(f.entries) {
		return nil, dialog.ErrCanceled
	}
	e := f.entries[f.callIdx]
	f.callIdx++
	return e.value, e.err
}

func (f *fakeDialog) ShowInfo(opts dialog.InfoOptions) error {
	f.infoTexts = append(f.infoTexts, opts.Text)
	return nil
}

// fakeLUKSDevice is a luksDevice test double. CheckKey returns valid only when
// validIn(slot, passphrase) reports true, simulating cryptsetup accepting the
// passphrase against a particular set of slots. It records every slot it was
// asked to check so tests can assert which slot the escrow flow used.
type fakeLUKSDevice struct {
	validIn      func(slot int, passphrase []byte) bool
	checkErr     error
	checkedSlots []int

	addErr            error
	addCalled         bool
	addExistingKey    *encryption.Key
	addNewKey         *encryption.Key
	addedSlots        map[int][]byte // slot -> passphrase, populated by AddKey
	dontRegisterAdded bool           // when true, AddKey succeeds but the key won't validate
}

func (d *fakeLUKSDevice) CheckKey(_ context.Context, _ string, key *encryption.Key) (bool, error) {
	d.checkedSlots = append(d.checkedSlots, key.Slot)
	if d.checkErr != nil {
		return false, d.checkErr
	}
	// A key just added by AddKey validates against its concrete slot.
	if pw, ok := d.addedSlots[key.Slot]; ok && string(pw) == string(key.Value) {
		return true, nil
	}
	if d.validIn == nil {
		return false, nil
	}
	return d.validIn(key.Slot, key.Value), nil
}

func (d *fakeLUKSDevice) AddKey(_ context.Context, _ string, key, newKey *encryption.Key) error {
	d.addCalled = true
	d.addExistingKey = key
	d.addNewKey = newKey
	if d.addErr != nil {
		return d.addErr
	}
	if d.dontRegisterAdded {
		return nil
	}
	if d.addedSlots == nil {
		d.addedSlots = make(map[int][]byte)
	}
	d.addedSlots[newKey.Slot] = newKey.Value
	return nil
}

// TestPromptAndValidatePassphraseValidatesAgainstAnySlot is the core
// regression test for issue #46227: a passphrase that only validates when no
// specific key slot is requested (i.e. the user's key lives in a non-zero
// slot) must still be accepted. The old code pinned the check to slot 0 and
// rejected such passphrases as if they were incorrect.
func TestPromptAndValidatePassphraseValidatesAgainstAnySlot(t *testing.T) {
	ctx := t.Context()
	correct := []byte("correct horse")

	dlg := &fakeDialog{entries: []scriptedEntry{{value: correct}}}
	dev := &fakeLUKSDevice{
		// Mimics cryptsetup behavior with the user's key in a non-zero slot:
		// rejected when --key-slot=0 is forced, accepted when any slot is allowed.
		validIn: func(slot int, passphrase []byte) bool {
			return slot == encryption.AnyKeyslot && string(passphrase) == string(correct)
		},
	}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda")
	require.NoError(t, err)
	assert.Equal(t, correct, got)

	// The passphrase must have been validated against any slot, not slot 0.
	require.Len(t, dev.checkedSlots, 1)
	assert.Equal(t, encryption.AnyKeyslot, dev.checkedSlots[0])
	// User was only prompted once, no retry.
	assert.Equal(t, []string{entryDialogText}, dlg.shownText)
}

// TestPromptAndValidatePassphraseRetries verifies that an incorrect passphrase
// re-prompts with the retry copy and that a subsequently correct passphrase is
// accepted.
func TestPromptAndValidatePassphraseRetries(t *testing.T) {
	ctx := t.Context()
	correct := []byte("right")

	dlg := &fakeDialog{entries: []scriptedEntry{
		{value: []byte("wrong")},
		{value: correct},
	}}
	dev := &fakeLUKSDevice{
		validIn: func(_ int, passphrase []byte) bool {
			return string(passphrase) == string(correct)
		},
	}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda")
	require.NoError(t, err)
	assert.Equal(t, correct, got)

	assert.Len(t, dev.checkedSlots, 2)
	// First prompt used the initial copy, second used the retry copy.
	assert.Equal(t, []string{entryDialogText, retryEntryDialogText}, dlg.shownText)
}

// TestPromptAndValidatePassphraseCanceled verifies that an empty entry (user
// canceled or the dialog timed out) returns a nil passphrase with no error and
// never attempts validation.
func TestPromptAndValidatePassphraseCanceled(t *testing.T) {
	ctx := t.Context()

	dlg := &fakeDialog{entries: []scriptedEntry{{value: nil}}}
	dev := &fakeLUKSDevice{}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Empty(t, dev.checkedSlots)
}

// TestPromptAndValidatePassphraseCanceledDuringRetry verifies that canceling
// at the retry prompt (after an incorrect first attempt) aborts cleanly.
func TestPromptAndValidatePassphraseCanceledDuringRetry(t *testing.T) {
	ctx := t.Context()

	dlg := &fakeDialog{entries: []scriptedEntry{
		{value: []byte("wrong")},
		{value: nil},
	}}
	dev := &fakeLUKSDevice{
		validIn: func(_ int, _ []byte) bool { return false },
	}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Len(t, dev.checkedSlots, 1)
}

// TestPromptAndValidatePassphraseCheckKeyError verifies that a genuine error
// from the device (as opposed to a rejected passphrase) is surfaced wrapped,
// rather than being treated as an incorrect passphrase.
func TestPromptAndValidatePassphraseCheckKeyError(t *testing.T) {
	ctx := t.Context()

	dlg := &fakeDialog{entries: []scriptedEntry{{value: []byte("whatever")}}}
	dev := &fakeLUKSDevice{checkErr: errors.New("cryptsetup boom")}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed validating passphrase")
	assert.Nil(t, got)
}

// TestPassphraseIsValidEmpty verifies the short-circuit: an empty passphrase is
// invalid without touching the device.
func TestPassphraseIsValidEmpty(t *testing.T) {
	ctx := t.Context()
	dev := &fakeLUKSDevice{}
	lr := &LuksRunner{}

	valid, err := lr.passphraseIsValid(ctx, dev, "/dev/sda", nil, encryption.AnyKeyslot)
	require.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, dev.checkedSlots)
}

// TestAddEscrowKeyUsesAnyKeyslotForExistingKey verifies that when adding the
// escrow key, the user's *existing* passphrase is presented with
// encryption.AnyKeyslot so cryptsetup finds whichever slot it lives in, while
// the new escrow key is pinned to the discovered free slot.
func TestAddEscrowKeyUsesAnyKeyslotForExistingKey(t *testing.T) {
	ctx := t.Context()
	userPassphrase := []byte("user secret in slot 3")
	escrowPassphrase := []byte("AAAA-BBBB-CCCC-DDDD")
	const escrowSlot uint = 4

	dev := &fakeLUKSDevice{}
	lr := &LuksRunner{}

	err := lr.addEscrowKey(ctx, dev, "/dev/sda", userPassphrase, escrowPassphrase, escrowSlot)
	require.NoError(t, err)

	require.True(t, dev.addCalled)
	// Existing key must not be pinned to a specific slot.
	require.NotNil(t, dev.addExistingKey)
	assert.Equal(t, encryption.AnyKeyslot, dev.addExistingKey.Slot)
	assert.Equal(t, userPassphrase, dev.addExistingKey.Value)
	// New escrow key must be pinned to the discovered free slot.
	require.NotNil(t, dev.addNewKey)
	assert.Equal(t, int(escrowSlot), dev.addNewKey.Slot)
	assert.Equal(t, escrowPassphrase, dev.addNewKey.Value)
	// Post-add validation checks the concrete escrow slot, not AnyKeyslot.
	assert.Equal(t, []int{int(escrowSlot)}, dev.checkedSlots)
}

// TestAddEscrowKeyValidationFails verifies that a freshly added key that does
// not validate surfaces an error rather than reporting success.
func TestAddEscrowKeyValidationFails(t *testing.T) {
	ctx := t.Context()
	dev := &fakeLUKSDevice{
		// AddKey succeeds but the key is never registered, so post-add
		// validation reports it invalid.
		dontRegisterAdded: true,
	}
	lr := &LuksRunner{}

	err := lr.addEscrowKey(ctx, dev, "/dev/sda", []byte("user"), []byte("escrow"), 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to validate escrow passphrase")
}

// TestAddEscrowKeyAddKeyError verifies that an error from device.AddKey is
// surfaced wrapped as "Failed to add key" and that no post-add validation is
// attempted.
func TestAddEscrowKeyAddKeyError(t *testing.T) {
	ctx := t.Context()
	dev := &fakeLUKSDevice{addErr: errors.New("cryptsetup add boom")}
	lr := &LuksRunner{}

	err := lr.addEscrowKey(ctx, dev, "/dev/sda", []byte("user"), []byte("escrow"), 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to add key")
	assert.Contains(t, err.Error(), "cryptsetup add boom")
	// AddKey failed, so the escrow key was never validated.
	assert.Empty(t, dev.checkedSlots)
}
