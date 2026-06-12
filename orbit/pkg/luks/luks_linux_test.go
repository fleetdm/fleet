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
}

func (d *fakeLUKSDevice) CheckKey(_ context.Context, _ string, key *encryption.Key) (bool, error) {
	d.checkedSlots = append(d.checkedSlots, key.Slot)
	if d.checkErr != nil {
		return false, d.checkErr
	}
	if d.validIn == nil {
		return false, nil
	}
	return d.validIn(key.Slot, key.Value), nil
}

func (d *fakeLUKSDevice) AddKey(_ context.Context, _ string, _, _ *encryption.Key) error {
	return nil
}

// TestPromptAndValidatePassphraseValidatesAgainstAnySlot is the core
// regression test for issue #46227: a passphrase that only validates when no
// specific key slot is requested (i.e. the user's key lives in a non-zero
// slot) must still be accepted. The old code pinned the check to slot 0 and
// rejected such passphrases as if they were incorrect.
func TestPromptAndValidatePassphraseValidatesAgainstAnySlot(t *testing.T) {
	ctx := context.Background()
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

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda", "title", "prompt", "retry")
	require.NoError(t, err)
	assert.Equal(t, correct, got)

	// The passphrase must have been validated against any slot, not slot 0.
	require.Len(t, dev.checkedSlots, 1)
	assert.Equal(t, encryption.AnyKeyslot, dev.checkedSlots[0])
	// User was only prompted once, no retry.
	assert.Equal(t, []string{"prompt"}, dlg.shownText)
}

// TestPromptAndValidatePassphraseRetries verifies that an incorrect passphrase
// re-prompts with the retry copy and that a subsequently correct passphrase is
// accepted.
func TestPromptAndValidatePassphraseRetries(t *testing.T) {
	ctx := context.Background()
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

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda", "title", "prompt", "retry")
	require.NoError(t, err)
	assert.Equal(t, correct, got)

	assert.Len(t, dev.checkedSlots, 2)
	// First prompt used the initial copy, second used the retry copy.
	assert.Equal(t, []string{"prompt", "retry"}, dlg.shownText)
}

// TestPromptAndValidatePassphraseCanceled verifies that an empty entry (user
// canceled or the dialog timed out) returns a nil passphrase with no error and
// never attempts validation.
func TestPromptAndValidatePassphraseCanceled(t *testing.T) {
	ctx := context.Background()

	dlg := &fakeDialog{entries: []scriptedEntry{{value: nil}}}
	dev := &fakeLUKSDevice{}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda", "title", "prompt", "retry")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Empty(t, dev.checkedSlots)
}

// TestPromptAndValidatePassphraseCanceledDuringRetry verifies that canceling
// at the retry prompt (after an incorrect first attempt) aborts cleanly.
func TestPromptAndValidatePassphraseCanceledDuringRetry(t *testing.T) {
	ctx := context.Background()

	dlg := &fakeDialog{entries: []scriptedEntry{
		{value: []byte("wrong")},
		{value: nil},
	}}
	dev := &fakeLUKSDevice{
		validIn: func(_ int, _ []byte) bool { return false },
	}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda", "title", "prompt", "retry")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Len(t, dev.checkedSlots, 1)
}

// TestPromptAndValidatePassphraseCheckKeyError verifies that a genuine error
// from the device (as opposed to a rejected passphrase) is surfaced wrapped,
// rather than being treated as an incorrect passphrase.
func TestPromptAndValidatePassphraseCheckKeyError(t *testing.T) {
	ctx := context.Background()

	dlg := &fakeDialog{entries: []scriptedEntry{{value: []byte("whatever")}}}
	dev := &fakeLUKSDevice{checkErr: errors.New("cryptsetup boom")}
	lr := &LuksRunner{notifier: dlg}

	got, err := lr.promptAndValidatePassphrase(ctx, dev, "/dev/sda", "title", "prompt", "retry")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed validating passphrase")
	assert.Nil(t, got)
}

// TestPassphraseIsValidEmpty verifies the short-circuit: an empty passphrase is
// invalid without touching the device.
func TestPassphraseIsValidEmpty(t *testing.T) {
	ctx := context.Background()
	dev := &fakeLUKSDevice{}
	lr := &LuksRunner{}

	valid, err := lr.passphraseIsValid(ctx, dev, "/dev/sda", nil, encryption.AnyKeyslot)
	require.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, dev.checkedSlots)
}
