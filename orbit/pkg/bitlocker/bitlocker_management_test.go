package bitlocker

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// A TPM-only protector unseals the volume with no user input, so adding one to a volume that already has a TPM+PIN
// protector silently removes pre-boot authentication.
func TestEnsureBootUnsealProtector(t *testing.T) {
	listErr := errors.New("WMI unavailable")

	for _, tc := range []struct {
		name       string
		hasBoot    bool
		hasBootErr error
		wantAdd    bool
	}{
		{name: "no boot protector, so one is added", wantAdd: true},
		{name: "a boot protector already exists, so none is added", hasBoot: true},
		// Adding one blind risks the bypass above, so an unreadable list is the safer failure.
		{name: "an unreadable protector list adds nothing", hasBootErr: listErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var added bool
			err := ensureBootUnsealProtector(
				func() (bool, error) { return tc.hasBoot, tc.hasBootErr },
				func() error { added = true; return nil },
			)

			require.Equal(t, tc.wantAdd, added)
			if tc.hasBootErr != nil {
				require.ErrorIs(t, err, tc.hasBootErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// fakePINVolume is an in-memory protector list for setTPMAndPINProtector.
type fakePINVolume struct {
	status     *EncryptionStatus
	statusErr  error
	protectors map[int32][]string
	// listErrAfterAdd fails listing protectors of a type once the PIN protector has been added.
	listErrAfterAdd map[int32]error
	addErr          error
	// dropsTPMOnly mimics Windows replacing the TPM-only protector itself when the PIN protector is added.
	dropsTPMOnly  bool
	deleteErr     map[string]error
	restoreTPMErr error

	pinAdded bool
	gotPIN   string
}

const (
	fakeTPMOnlyID         = "{tpm}"
	fakeRecoveryID        = "{recovery}"
	fakePINID             = "{tpm-and-pin}"
	fakeRestoredTPMOnlyID = "{restored-tpm}"
)

func (f *fakePINVolume) getBitlockerStatus() (*EncryptionStatus, error) {
	return f.status, f.statusErr
}

func (f *fakePINVolume) getKeyProtectorIDs(protectorType int32) ([]string, error) {
	if err := f.listErrAfterAdd[protectorType]; err != nil && f.pinAdded {
		return nil, err
	}
	return slices.Clone(f.protectors[protectorType]), nil
}

func (f *fakePINVolume) protectWithTPMAndPIN(pin string) (string, error) {
	f.gotPIN = pin
	if f.addErr != nil {
		return "", f.addErr
	}
	f.pinAdded = true
	f.protectors[KeyProtectorTypeTPMAndPIN] = append(f.protectors[KeyProtectorTypeTPMAndPIN], fakePINID)
	if f.dropsTPMOnly {
		delete(f.protectors, KeyProtectorTypeTPM)
	}
	return fakePINID, nil
}

func (f *fakePINVolume) protectWithTPM(*[]uint8) error {
	if f.restoreTPMErr != nil {
		return f.restoreTPMErr
	}
	if len(f.protectors[KeyProtectorTypeTPM]) > 0 {
		return NewEncryptionError("exists", ErrorCodeProtectorExists)
	}
	f.protectors[KeyProtectorTypeTPM] = []string{fakeRestoredTPMOnlyID}
	return nil
}

func (f *fakePINVolume) deleteKeyProtector(protectorID string) error {
	if err := f.deleteErr[protectorID]; err != nil {
		return err
	}
	for protectorType, ids := range f.protectors {
		f.protectors[protectorType] = slices.DeleteFunc(ids, func(id string) bool { return id == protectorID })
		if len(f.protectors[protectorType]) == 0 {
			delete(f.protectors, protectorType)
		}
	}
	return nil
}

func TestSetTPMAndPINProtector(t *testing.T) {
	t.Parallel()

	// An enhanced PIN with the characters most likely to be mangled on the way to WMI.
	const pin = ` my "PIN" \ 1'2 `

	protectedAndEncrypted := &EncryptionStatus{ConversionStatus: ConversionStatusFullyEncrypted, ProtectionStatus: ProtectionStatusOn}
	tpmOnlyAndRecovery := func() map[int32][]string {
		return map[int32][]string{
			KeyProtectorTypeTPM:               {fakeTPMOnlyID},
			KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
		}
	}
	pinAndRecovery := map[int32][]string{
		KeyProtectorTypeTPMAndPIN:         {fakePINID},
		KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
	}
	wmiErr := errors.New("WMI unavailable")

	for _, tc := range []struct {
		name           string
		vol            *fakePINVolume
		wantReason     string
		wantAddCalled  bool
		wantProtectors map[int32][]string
	}{
		{
			name:           "adds the PIN and removes the TPM-only protector",
			vol:            &fakePINVolume{status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery()},
			wantAddCalled:  true,
			wantProtectors: pinAndRecovery,
		},
		{
			name:           "Windows already replaced the TPM-only protector",
			vol:            &fakePINVolume{status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(), dropsTPMOnly: true},
			wantAddCalled:  true,
			wantProtectors: pinAndRecovery,
		},
		{
			name: "a PIN is already set",
			vol: &fakePINVolume{status: protectedAndEncrypted, protectors: map[int32][]string{
				KeyProtectorTypeTPMAndPIN: {"{existing}"}, KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
			}},
			wantReason: PINReasonAlreadySet,
			wantProtectors: map[int32][]string{
				KeyProtectorTypeTPMAndPIN: {"{existing}"}, KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
			},
		},
		{
			name: "a PIN and startup key is already set",
			vol: &fakePINVolume{status: protectedAndEncrypted, protectors: map[int32][]string{
				KeyProtectorTypeTPMAndPINAndStartupKey: {"{existing}"},
			}},
			wantReason:     PINReasonAlreadySet,
			wantProtectors: map[int32][]string{KeyProtectorTypeTPMAndPINAndStartupKey: {"{existing}"}},
		},
		{
			name: "protection is off",
			vol: &fakePINVolume{
				status:     &EncryptionStatus{ConversionStatus: ConversionStatusFullyEncrypted, ProtectionStatus: ProtectionStatusOff},
				protectors: tpmOnlyAndRecovery(),
			},
			wantReason:     PINReasonProtectionOff,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name: "not fully encrypted",
			vol: &fakePINVolume{
				status:     &EncryptionStatus{ConversionStatus: ConversionStatusEncryptionInProgress, ProtectionStatus: ProtectionStatusOn},
				protectors: tpmOnlyAndRecovery(),
			},
			wantReason:     PINReasonNotFullyEncrypted,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name:           "status is unreadable",
			vol:            &fakePINVolume{statusErr: wmiErr, protectors: tpmOnlyAndRecovery()},
			wantReason:     PINReasonStatusUnreadable,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name: "Windows rejects the PIN characters",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(),
				addErr: NewEncryptionError("invalid chars", ErrorCodeInvalidPINChars),
			},
			wantReason:     PINReasonInvalidChars,
			wantAddCalled:  true,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name: "Windows says a PIN protector exists",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(),
				addErr: NewEncryptionError("exists", ErrorCodeProtectorExists),
			},
			wantReason:     PINReasonAlreadySet,
			wantAddCalled:  true,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name: "deleting the TPM-only protector fails, so the PIN is rolled back",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(),
				deleteErr: map[string]error{fakeTPMOnlyID: wmiErr},
			},
			wantReason:     PINReasonNotFinished,
			wantAddCalled:  true,
			wantProtectors: tpmOnlyAndRecovery(),
		},
		{
			name: "confirming the PIN fails after Windows dropped the TPM-only protector, so it is restored",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(), dropsTPMOnly: true,
				listErrAfterAdd: map[int32]error{KeyProtectorTypeTPMAndPIN: wmiErr},
			},
			wantReason:    PINReasonNotFinished,
			wantAddCalled: true,
			wantProtectors: map[int32][]string{
				KeyProtectorTypeTPM:               {fakeRestoredTPMOnlyID},
				KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
			},
		},
		{
			// The volume is left with only a recovery password. Retrying succeeds, because nothing blocks adding the PIN.
			name: "the rollback cannot restore the TPM-only protector Windows dropped",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(), dropsTPMOnly: true,
				listErrAfterAdd: map[int32]error{KeyProtectorTypeTPMAndPIN: wmiErr},
				restoreTPMErr:   NewEncryptionError("TBS stopped", ErrorCodeTBSServiceNotRunning),
			},
			wantReason:     PINReasonNotFinished,
			wantAddCalled:  true,
			wantProtectors: map[int32][]string{KeyProtectorTypeNumericalPassword: {fakeRecoveryID}},
		},
		{
			name: "the rollback fails too",
			vol: &fakePINVolume{
				status: protectedAndEncrypted, protectors: tpmOnlyAndRecovery(),
				deleteErr: map[string]error{fakeTPMOnlyID: wmiErr, fakePINID: wmiErr},
			},
			wantReason:    PINReasonNotFinished,
			wantAddCalled: true,
			wantProtectors: map[int32][]string{
				KeyProtectorTypeTPM:               {fakeTPMOnlyID},
				KeyProtectorTypeTPMAndPIN:         {fakePINID},
				KeyProtectorTypeNumericalPassword: {fakeRecoveryID},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := setTPMAndPINProtector(tc.vol, pin)

			if tc.wantReason == "" {
				require.NoError(t, err)
			} else {
				pinErr, ok := errors.AsType[*PINError](err)
				require.True(t, ok, "want a *PINError, got %v", err)
				require.Equal(t, tc.wantReason, pinErr.Reason)
				require.NotContains(t, err.Error(), pin)
			}
			if tc.wantAddCalled {
				require.Equal(t, pin, tc.vol.gotPIN)
			} else {
				require.Empty(t, tc.vol.gotPIN)
			}
			require.Equal(t, tc.wantProtectors, tc.vol.protectors)
		})
	}
}

func TestPINAddFailureReason(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{name: "PIN length", err: NewEncryptionError("", ErrorCodeInvalidPINLength), want: PINReasonInvalidLength},
		{name: "PIN characters", err: NewEncryptionError("", ErrorCodeInvalidPINCharsDetailed), want: PINReasonInvalidChars},
		{name: "TPM service", err: NewEncryptionError("", ErrorCodeTBSServiceNotRunning), want: PINReasonTPMServiceStopped},
		{name: "locked volume", err: NewEncryptionError("", ErrorCodeLockedVolume), want: PINReasonLockedVolume},
		{name: "bootable media", err: NewEncryptionError("", ErrorCodeBootableCDOrDVD), want: PINReasonBootableMedia},
		{name: "foreign volume", err: NewEncryptionError("", ErrorCodeForeignVolume), want: PINReasonForeignVolume},
		{name: "unmapped code", err: NewEncryptionError("", ErrorCodeIODevice), want: "Windows couldn't add the PIN (error 0x8007045D)"},
		{name: "not a WMI error", err: errors.New("COM failure"), want: PINReasonNotFinished},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, pinAddFailureReason(fmt.Errorf("wrapped: %w", tc.err)))
		})
	}
}
