package bitlocker

import (
	"errors"
	"fmt"
	"slices"
)

// Volume encryption/decryption status.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
const (
	ConversionStatusFullyDecrypted       int32 = 0
	ConversionStatusFullyEncrypted       int32 = 1
	ConversionStatusEncryptionInProgress int32 = 2
	ConversionStatusDecryptionInProgress int32 = 3
	ConversionStatusEncryptionPaused     int32 = 4
	ConversionStatusDecryptionPaused     int32 = 5
)

const (
	// Error codes from Win32_EncryptableVolume WMI methods. The Microsoft docs
	// define these as uint32, but the COM VARIANT transport delivers them as
	// VT_I4 (signed 32-bit), which go-ole surfaces as int32. The bit patterns
	// are identical (e.g., 0x80310019 as uint32 == -2144272327 as int32).
	ErrorCodeInvalidArg                 int32 = -2147024809 // E_INVALIDARG: encryption flags conflict with Group Policy
	ErrorCodeIODevice                   int32 = -2147023779
	ErrorCodeDriveIncompatibleVolume    int32 = -2144272206
	ErrorCodeNoTPMWithPassphrase        int32 = -2144272212
	ErrorCodePassphraseTooLong          int32 = -2144272214
	ErrorCodePolicyPassphraseNotAllowed int32 = -2144272278
	ErrorCodeNotDecrypted               int32 = -2144272327
	ErrorCodeInvalidPasswordFormat      int32 = -2144272331
	ErrorCodeBootableCDOrDVD            int32 = -2144272336
	ErrorCodeProtectorExists            int32 = -2144272335
	ErrorCodeInvalidPINLength           int32 = -2144272280 // FVE_E_POLICY_INVALID_PIN_LENGTH 0x80310068
	ErrorCodeInvalidPINChars            int32 = -2144272230 // FVE_E_INVALID_PIN_CHARS 0x8031009A
	ErrorCodeInvalidPINCharsDetailed    int32 = -2144272180 // FVE_E_INVALID_PIN_CHARS_DETAILED 0x803100CC, what current Windows returns
	ErrorCodeTBSServiceNotRunning       int32 = -2144845816 // TBS_E_SERVICE_NOT_RUNNING 0x80284008
	ErrorCodeLockedVolume               int32 = -2144272384 // FVE_E_LOCKED_VOLUME 0x80310000
	ErrorCodeForeignVolume              int32 = -2144272349 // FVE_E_FOREIGN_VOLUME 0x80310023
)

// EncryptionError represents an error that occurs during the encryption
// process.
type EncryptionError struct {
	msg  string // msg is the error message describing what went wrong.
	code int32  // code is the Bitlocker-specific error code.
}

func NewEncryptionError(msg string, code int32) *EncryptionError {
	return &EncryptionError{
		msg:  msg,
		code: code,
	}
}

// Error returns the error message of the EncryptionError.
// This method makes EncryptionError compatible with the Go built-in error
// interface.
func (e *EncryptionError) Error() string {
	return e.msg
}

// Code returns the Bitlocker-specific error code.
// These codes are defined by Microsoft and are used to identify specific types
// of encryption errors.
func (e *EncryptionError) Code() int32 {
	return e.code
}

// EncryptionStatus represents the encryption status of a volume as returned by
// the GetConversionStatus method of the Win32_EncryptableVolume class.
type EncryptionStatus struct {
	ProtectionStatus     int32  // indicates whether the volume and its encryption key are secured.
	ConversionStatus     int32  // represents the encryption or decryption status of the volume.
	EncryptionPercentage string // percentage of the volume that is encrypted.
	EncryptionFlags      string // flags describing the encryption behavior.
	WipingStatus         int32  // status of the free space wiping on the volume.
	WipingPercentage     string // percentage of free space that has been wiped.
}

// VolumeStatus provides the encryption status for a specific drive volume.
// It ties a volume (identified by its drive letter) to its EncryptionStatus.
type VolumeStatus struct {
	DriveVolume string            // driveVolume is the identifier of the drive (e.g., "C:").
	Status      *EncryptionStatus // status holds the encryption status of the volume.
	// Err is set when the status for this volume could not be read. A volume whose status could not be read must never be treated as
	// "not encrypted".
	Err error
}

// Volume protection status, as returned by GetProtectionStatus.
// https://learn.microsoft.com/en-us/windows/win32/secprov/getprotectionstatus-win32-encryptablevolume
const (
	ProtectionStatusOff int32 = 0
	ProtectionStatusOn  int32 = 1
)

// Key protector types for GetKeyProtectors.
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectors-win32-encryptablevolume
const (
	KeyProtectorTypeTPM                    int32 = 1
	KeyProtectorTypeExternalKey            int32 = 2
	KeyProtectorTypeNumericalPassword      int32 = 3
	KeyProtectorTypeTPMAndPIN              int32 = 4
	KeyProtectorTypeTPMAndStartupKey       int32 = 5
	KeyProtectorTypeTPMAndPINAndStartupKey int32 = 6
)

// BootUnsealProtectorTypes are the key protector types that can release the volume master key at boot without a human typing the
// 48-digit recovery password. The PIN and startup key variants prompt the user, which is by design and is not a recovery prompt.
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectors-win32-encryptablevolume
var BootUnsealProtectorTypes = []int32{
	KeyProtectorTypeTPM,
	KeyProtectorTypeExternalKey,
	KeyProtectorTypeTPMAndPIN,
	KeyProtectorTypeTPMAndStartupKey,
	KeyProtectorTypeTPMAndPINAndStartupKey,
}

// ensureBootUnsealProtector adds a TPM-only protector when, and only when, the volume has nothing that can already
// release the volume master key at boot.
func ensureBootUnsealProtector(hasBootProtector func() (bool, error), addTPMProtector func() error) error {
	has, err := hasBootProtector()
	if err != nil {
		// Adding a protector blind risks the bypass above, so leaving a pre-encrypted disk without a TPM protector is
		// the safer of the two failures.
		return fmt.Errorf("listing boot protectors: %w", err)
	}
	if has {
		return nil
	}
	return addTPMProtector()
}

// Reasons a startup PIN could not be set. The My device page shows them as "Couldn't set PIN. {reason}. Try again or
// contact your IT admin.", so each is a short sentence-case phrase without a trailing period.
const (
	PINReasonAlreadySet        = "PIN already set"
	PINReasonWindowsServer     = "Windows Server isn't supported"
	PINReasonStatusUnreadable  = "Couldn't read this device's BitLocker status"
	PINReasonNotFullyEncrypted = "Disk encryption isn't finished yet"
	PINReasonProtectionOff     = "BitLocker protection is suspended"
	PINReasonInvalidLength     = "This device's BitLocker policy doesn't allow a PIN of that length"
	PINReasonInvalidChars      = "This device's BitLocker policy doesn't allow those characters"
	PINReasonTPMServiceStopped = "The TPM service isn't running"
	PINReasonLockedVolume      = "The drive is locked"
	PINReasonBootableMedia     = "Remove the CD or DVD from the drive"
	PINReasonForeignVolume     = "The drive doesn't contain the running copy of Windows"
	PINReasonNotFinished       = "Windows couldn't finish setting the PIN"
)

// PINError is a failure to set a startup PIN. It never contains the PIN.
type PINError struct {
	// Reason is safe to show the end user; see the PINReason constants.
	Reason string
	Err    error
}

func (e *PINError) Error() string {
	if e.Err == nil {
		return e.Reason
	}
	return e.Reason + ": " + e.Err.Error()
}

func (e *PINError) Unwrap() error {
	return e.Err
}

// pinAddFailureReason explains why Windows refused to add a TPM and PIN protector.
func pinAddFailureReason(err error) string {
	encErr, ok := errors.AsType[*EncryptionError](err)
	if !ok {
		return PINReasonNotFinished
	}
	switch encErr.Code() {
	case ErrorCodeInvalidPINLength:
		return PINReasonInvalidLength
	case ErrorCodeInvalidPINChars, ErrorCodeInvalidPINCharsDetailed:
		return PINReasonInvalidChars
	case ErrorCodeProtectorExists:
		// Never success: the PIN the end user typed is not the one on the volume.
		return PINReasonAlreadySet
	case ErrorCodeTBSServiceNotRunning:
		return PINReasonTPMServiceStopped
	case ErrorCodeLockedVolume:
		return PINReasonLockedVolume
	case ErrorCodeBootableCDOrDVD:
		return PINReasonBootableMedia
	case ErrorCodeForeignVolume:
		return PINReasonForeignVolume
	default:
		return fmt.Sprintf("Windows couldn't add the PIN (error 0x%08X)", uint32(encErr.Code())) // nolint:gosec
	}
}

// pinProtectorVolume is the part of a BitLocker volume that setTPMAndPINProtector uses, so the sequence can be tested
// without COM.
type pinProtectorVolume interface {
	getBitlockerStatus() (*EncryptionStatus, error)
	getKeyProtectorIDs(protectorType int32) ([]string, error)
	protectWithTPMAndPIN(pin string) (protectorID string, err error)
	protectWithTPM(platformValidationProfile *[]uint8) error
	deleteKeyProtector(protectorID string) error
}

// setTPMAndPINProtector adds a TPM and PIN protector to a protected, fully encrypted volume and removes its TPM-only
// protectors, which would otherwise unseal the volume at boot without the PIN. If anything fails after the add, the new
// protector is removed so the volume is left as it was found.
func setTPMAndPINProtector(vol pinProtectorVolume, pin string) error {
	status, err := vol.getBitlockerStatus()
	if err != nil {
		return &PINError{Reason: PINReasonStatusUnreadable, Err: err}
	}
	if status.ConversionStatus != ConversionStatusFullyEncrypted {
		return &PINError{Reason: PINReasonNotFullyEncrypted, Err: fmt.Errorf("conversion status %d", status.ConversionStatus)}
	}
	if status.ProtectionStatus != ProtectionStatusOn {
		return &PINError{Reason: PINReasonProtectionOff, Err: fmt.Errorf("protection status %d", status.ProtectionStatus)}
	}
	for _, protectorType := range []int32{KeyProtectorTypeTPMAndPIN, KeyProtectorTypeTPMAndPINAndStartupKey} {
		ids, err := vol.getKeyProtectorIDs(protectorType)
		if err != nil {
			return &PINError{Reason: PINReasonStatusUnreadable, Err: err}
		}
		if len(ids) > 0 {
			return &PINError{Reason: PINReasonAlreadySet, Err: fmt.Errorf("a protector of type %d exists", protectorType)}
		}
	}
	tpmOnlyIDs, err := vol.getKeyProtectorIDs(KeyProtectorTypeTPM)
	if err != nil {
		return &PINError{Reason: PINReasonStatusUnreadable, Err: err}
	}

	pinProtectorID, err := vol.protectWithTPMAndPIN(pin)
	if err != nil {
		return &PINError{Reason: pinAddFailureReason(err), Err: fmt.Errorf("adding the TPM and PIN protector: %w", err)}
	}

	if err := removeTPMOnlyProtectors(vol, pinProtectorID); err != nil {
		if rollbackErr := rollBackTPMAndPINProtector(vol, pinProtectorID, len(tpmOnlyIDs) > 0); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("rolling back the TPM and PIN protector: %w", rollbackErr))
		}
		return &PINError{Reason: PINReasonNotFinished, Err: err}
	}
	return nil
}

func removeTPMOnlyProtectors(vol pinProtectorVolume, pinProtectorID string) error {
	pinIDs, err := vol.getKeyProtectorIDs(KeyProtectorTypeTPMAndPIN)
	if err != nil {
		return fmt.Errorf("confirming the TPM and PIN protector: %w", err)
	}
	if !slices.Contains(pinIDs, pinProtectorID) {
		return fmt.Errorf("the TPM and PIN protector %s is missing after adding it", pinProtectorID)
	}

	tpmOnlyIDs, err := vol.getKeyProtectorIDs(KeyProtectorTypeTPM)
	if err != nil {
		return fmt.Errorf("listing TPM-only protectors: %w", err)
	}
	for _, id := range tpmOnlyIDs {
		if err := vol.deleteKeyProtector(id); err != nil {
			return fmt.Errorf("deleting TPM-only protector: %w", err)
		}
	}

	remaining, err := vol.getKeyProtectorIDs(KeyProtectorTypeTPM)
	if err != nil {
		return fmt.Errorf("confirming TPM-only protectors are gone: %w", err)
	}
	if len(remaining) > 0 {
		return fmt.Errorf("%d TPM-only protectors remain after deleting them", len(remaining))
	}
	return nil
}

// rollBackTPMAndPINProtector removes the protector setTPMAndPINProtector added. Windows may already have dropped the
// TPM-only protector when the PIN protector was added, and a volume left with neither boots to the recovery prompt, so a
// TPM-only protector the volume started with is put back.
func rollBackTPMAndPINProtector(vol pinProtectorVolume, pinProtectorID string, hadTPMOnly bool) error {
	var errs []error
	if err := vol.deleteKeyProtector(pinProtectorID); err != nil {
		errs = append(errs, err)
	}
	if hadTPMOnly {
		if err := vol.protectWithTPM(nil); err != nil {
			if encErr, ok := errors.AsType[*EncryptionError](err); !ok || encErr.Code() != ErrorCodeProtectorExists {
				errs = append(errs, fmt.Errorf("restoring the TPM-only protector: %w", err))
			}
		}
	}
	return errors.Join(errs...)
}
