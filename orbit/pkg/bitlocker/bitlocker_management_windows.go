//go:build windows

package bitlocker

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/registry"
)

// Encryption Methods
// https://docs.microsoft.com/en-us/windows/win32/secprov/getencryptionmethod-win32-encryptablevolume
type EncryptionMethod int32

const (
	None EncryptionMethod = iota
	AES128WithDiffuser
	AES256WithDiffuser
	AES128
	AES256
	HardwareEncryption
	XtsAES128
	XtsAES256
)

// Encryption Flags
// https://docs.microsoft.com/en-us/windows/win32/secprov/encrypt-win32-encryptablevolume
type EncryptionFlag int32

const (
	EncryptDataOnly EncryptionFlag = 0x00000001
	// EncryptDemandWipe encrypts the entire disk, including (wiping) free space.
	EncryptDemandWipe  EncryptionFlag = 0x00000002
	EncryptSynchronous EncryptionFlag = 0x00010000
)

// DiscoveryVolumeType specifies the type of discovery volume to be used by Prepare.
// https://docs.microsoft.com/en-us/windows/win32/secprov/preparevolume-win32-encryptablevolume
type DiscoveryVolumeType string

const (
	// VolumeTypeNone indicates no discovery volume. This value creates a native BitLocker volume.
	VolumeTypeNone DiscoveryVolumeType = "<none>"
	// VolumeTypeDefault indicates the default behavior.
	VolumeTypeDefault DiscoveryVolumeType = "<default>"
	// VolumeTypeFAT32 creates a FAT32 discovery volume.
	VolumeTypeFAT32 DiscoveryVolumeType = "FAT32"
)

// ForceEncryptionType specifies the encryption type to be used when calling Prepare on the volume.
// https://docs.microsoft.com/en-us/windows/win32/secprov/preparevolume-win32-encryptablevolume
type ForceEncryptionType int32

const (
	// EncryptionTypeUnspecified indicates that the encryption type is not specified.
	EncryptionTypeUnspecified ForceEncryptionType = 0
	// EncryptionTypeSoftware specifies software encryption.
	EncryptionTypeSoftware ForceEncryptionType = 1
	// EncryptionTypeHardware specifies hardware encryption.
	EncryptionTypeHardware ForceEncryptionType = 2
)

// fveErrorCode formats a BitLocker error code as "unsigned_decimal (0xHEX)" to
// match the format used in the Microsoft WMI documentation, making errors
// searchable. The WMI docs define return values as uint32 but the COM VARIANT
// transport delivers them as int32 (see comment on the error code constants).
func fveErrorCode(val int32) string {
	return fmt.Sprintf("%d (0x%08x)", uint32(val), uint32(val)) // nolint:gosec
}

func encryptErrHandler(val int32) error {
	var msg string

	switch val {
	case ErrorCodeInvalidArg:
		msg = "the encryption flags conflict with the current Group Policy settings (check HKLM\\SOFTWARE\\Policies\\Microsoft\\FVE)"
	case ErrorCodeIODevice:
		msg = "an I/O error has occurred during encryption; the device may need to be reset"
	case ErrorCodeDriveIncompatibleVolume:
		msg = "the drive specified does not support hardware-based encryption"
	case ErrorCodeNoTPMWithPassphrase:
		msg = "a TPM key protector cannot be added because a password protector exists on the drive"
	case ErrorCodePassphraseTooLong:
		msg = "the passphrase cannot exceed 256 characters"
	case ErrorCodePolicyPassphraseNotAllowed:
		msg = "group Policy settings do not permit the creation of a password"
	case ErrorCodeNotDecrypted:
		msg = "the drive must be fully decrypted to complete this operation"
	case ErrorCodeInvalidPasswordFormat:
		msg = "the format of the recovery password provided is invalid"
	case ErrorCodeBootableCDOrDVD:
		msg = "BitLocker Drive Encryption detected bootable media (CD or DVD) in the computer"
	case ErrorCodeProtectorExists:
		msg = "key protector cannot be added; only one key protector of this type is allowed for this drive"
	default:
		msg = fmt.Sprintf("error code returned during encryption: %s", fveErrorCode(val))
	}

	return &EncryptionError{msg, val}
}

/////////////////////////////////////////////////////
// Volume represents a Bitlocker encryptable volume
/////////////////////////////////////////////////////

type Volume struct {
	letter  string
	handle  *ole.IDispatch
	wmiIntf *ole.IDispatch
	wmiSvc  *ole.IDispatch
}

// bitlockerClose frees all resources associated with a volume.
func (v *Volume) bitlockerClose() {
	if v.handle != nil {
		v.handle.Release()
	}

	if v.wmiIntf != nil {
		v.wmiIntf.Release()
	}

	if v.wmiSvc != nil {
		v.wmiSvc.Release()
	}

}

// encrypt encrypts the volume
// Example: vol.encrypt(bitlocker.XtsAES256, bitlocker.EncryptDataOnly)
// https://docs.microsoft.com/en-us/windows/win32/secprov/encrypt-win32-encryptablevolume
func (v *Volume) encrypt(method EncryptionMethod, flags EncryptionFlag) error {
	resultRaw, err := oleutil.CallMethod(v.handle, "Encrypt", int32(method), int32(flags))
	if err != nil {
		return fmt.Errorf("encrypt(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("encrypt(%s): %w", v.letter, encryptErrHandler(val))
	}

	return nil
}

// prepareVolume prepares a new Bitlocker Volume. This should be called BEFORE any key protectors are added.
// Example: vol.prepareVolume(bitlocker.VolumeTypeDefault, bitlocker.EncryptionTypeHardware)
// https://docs.microsoft.com/en-us/windows/win32/secprov/preparevolume-win32-encryptablevolume
func (v *Volume) prepareVolume(volType DiscoveryVolumeType, encType ForceEncryptionType) error {
	resultRaw, err := oleutil.CallMethod(v.handle, "PrepareVolume", string(volType), int32(encType))
	if err != nil {
		return fmt.Errorf("prepareVolume(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("prepareVolume(%s): %w", v.letter, encryptErrHandler(val))
	}
	return nil
}

// protectWithNumericalPassword adds a numerical password key protector.
// Leave password as a blank string to have one auto-generated by Windows
// https://docs.microsoft.com/en-us/windows/win32/secprov/protectkeywithnumericalpassword-win32-encryptablevolume
func (v *Volume) protectWithNumericalPassword() (string, error) {
	var volumeKeyProtectorID ole.VARIANT
	_ = ole.VariantInit(&volumeKeyProtectorID)
	var resultRaw *ole.VARIANT
	var err error

	resultRaw, err = oleutil.CallMethod(v.handle, "ProtectKeyWithNumericalPassword", nil, nil, &volumeKeyProtectorID)
	if err != nil {
		return "", fmt.Errorf("ProtectKeyWithNumericalPassword(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return "", fmt.Errorf("ProtectKeyWithNumericalPassword(%s): %w", v.letter, encryptErrHandler(val))
	}

	var recoveryKey ole.VARIANT
	_ = ole.VariantInit(&recoveryKey)
	resultRaw, err = oleutil.CallMethod(v.handle, "GetKeyProtectorNumericalPassword", volumeKeyProtectorID.ToString(), &recoveryKey)

	if err != nil {
		return "", fmt.Errorf("GetKeyProtectorNumericalPassword(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return "", fmt.Errorf("GetKeyProtectorNumericalPassword(%s): %w", v.letter, encryptErrHandler(val))
	}

	return recoveryKey.ToString(), nil
}

// protectWithTPM adds the TPM key protector
// https://docs.microsoft.com/en-us/windows/win32/secprov/protectkeywithtpm-win32-encryptablevolume
func (v *Volume) protectWithTPM(platformValidationProfile *[]uint8) error {
	var volumeKeyProtectorID ole.VARIANT
	_ = ole.VariantInit(&volumeKeyProtectorID)
	var resultRaw *ole.VARIANT
	var err error

	if platformValidationProfile == nil {
		resultRaw, err = oleutil.CallMethod(v.handle, "ProtectKeyWithTPM", nil, nil, &volumeKeyProtectorID)
	} else {
		resultRaw, err = oleutil.CallMethod(v.handle, "ProtectKeyWithTPM", nil, *platformValidationProfile, &volumeKeyProtectorID)
	}
	if err != nil {
		return fmt.Errorf("protectKeyWithTPM(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("protectKeyWithTPM(%s): %w", v.letter, encryptErrHandler(val))
	}

	return nil
}

// deleteKeyProtectors removes all key protectors from the volume.
// https://learn.microsoft.com/en-us/windows/win32/secprov/deletekeyprotectors-win32-encryptablevolume
func (v *Volume) deleteKeyProtectors() error {
	resultRaw, err := oleutil.CallMethod(v.handle, "DeleteKeyProtectors")
	if err != nil {
		return fmt.Errorf("deleteKeyProtectors(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("deleteKeyProtectors(%s): %w", v.letter, encryptErrHandler(val))
	}
	return nil
}

// deleteKeyProtector removes a single key protector by its ID.
// https://learn.microsoft.com/en-us/windows/win32/secprov/deletekeyprotector-win32-encryptablevolume
func (v *Volume) deleteKeyProtector(protectorID string) error {
	resultRaw, err := oleutil.CallMethod(v.handle, "DeleteKeyProtector", protectorID)
	if err != nil {
		return fmt.Errorf("deleteKeyProtector(%s, %s): %w", v.letter, protectorID, err)
	}
	if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("deleteKeyProtector(%s, %s): %w", v.letter, protectorID, encryptErrHandler(val))
	}
	return nil
}

// Key protector types for GetKeyProtectors.
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectors-win32-encryptablevolume
const (
	KeyProtectorTypeNumericalPassword int32 = 3
)

// getKeyProtectorIDs returns the IDs of key protectors of the given type.
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectors-win32-encryptablevolume
func (v *Volume) getKeyProtectorIDs(protectorType int32) ([]string, error) {
	var protectorIDs ole.VARIANT
	_ = ole.VariantInit(&protectorIDs)
	defer ole.VariantClear(&protectorIDs) //nolint:errcheck

	resultRaw, err := oleutil.CallMethod(v.handle, "GetKeyProtectors", protectorType, &protectorIDs)
	if err != nil {
		return nil, fmt.Errorf("getKeyProtectors(%s, %d): %w", v.letter, protectorType, err)
	}
	if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return nil, fmt.Errorf("getKeyProtectors(%s, %d): %w", v.letter, protectorType, encryptErrHandler(val))
	}

	// The WMI method returns an out-parameter VARIANT containing a SAFEARRAY.
	// The array type is VT_ARRAY|VT_VARIANT (0x200C), not VT_ARRAY|VT_BSTR.
	// We use ToValueArray() to extract each element as an interface{}, then
	// convert to strings. We do NOT call safeArray.Release() here because
	// ToArray() wraps the same pointer from the VARIANT without copying --
	// defer VariantClear above handles freeing the SAFEARRAY.
	safeArray := protectorIDs.ToArray()
	if safeArray == nil {
		return nil, nil
	}

	values := safeArray.ToValueArray()
	result := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result, nil
}

// getBitlockerStatus returns the current status of the volume
// https://learn.microsoft.com/en-us/windows/win32/secprov/getprotectionstatus-win32-encryptablevolume
func (v *Volume) getBitlockerStatus() (*EncryptionStatus, error) {
	var (
		conversionStatus     int32
		encryptionPercentage int32
		encryptionFlags      int32
		wipingStatus         int32
		wipingPercentage     int32
		precisionFactor      int32 = 4
		protectionStatus     int32
	)

	resultRaw, err := oleutil.CallMethod(v.handle, "GetConversionStatus", &conversionStatus, &encryptionPercentage, &encryptionFlags, &wipingStatus, &wipingPercentage, precisionFactor)
	if err != nil {
		return nil, fmt.Errorf("GetConversionStatus(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return nil, fmt.Errorf("GetConversionStatus(%s): %w", v.letter, encryptErrHandler(val))
	}

	resultRaw, err = oleutil.CallMethod(v.handle, "GetProtectionStatus", &protectionStatus)
	if err != nil {
		return nil, fmt.Errorf("GetProtectionStatus(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return nil, fmt.Errorf("GetProtectionStatus(%s): %w", v.letter, encryptErrHandler(val))
	}

	// Creating the encryption status struct
	encStatus := &EncryptionStatus{
		ProtectionStatus:     protectionStatus,
		ConversionStatus:     conversionStatus,
		EncryptionPercentage: intToPercentage(encryptionPercentage),
		EncryptionFlags:      fmt.Sprintf("%d", encryptionFlags),
		WipingStatus:         wipingStatus,
		WipingPercentage:     intToPercentage(wipingPercentage),
	}

	return encStatus, nil
}

/////////////////////////////////////////////////////
// Helper functions
/////////////////////////////////////////////////////

// bitlockerConnect connects to an encryptable volume in order to manage it.
func bitlockerConnect(driveLetter string) (Volume, error) {
	v := Volume{letter: driveLetter}

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return v, fmt.Errorf("createObject: %w", err)
	}
	defer unknown.Release()

	v.wmiIntf, err = unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return v, fmt.Errorf("queryInterface: %w", err)
	}
	serviceRaw, err := oleutil.CallMethod(v.wmiIntf, "ConnectServer", nil, `\\.\ROOT\CIMV2\Security\MicrosoftVolumeEncryption`)
	if err != nil {
		v.bitlockerClose()
		return v, fmt.Errorf("connectServer: %w", err)
	}
	v.wmiSvc = serviceRaw.ToIDispatch()

	raw, err := oleutil.CallMethod(v.wmiSvc, "ExecQuery", "SELECT * FROM Win32_EncryptableVolume WHERE DriveLetter = '"+driveLetter+"'")
	if err != nil {
		v.bitlockerClose()
		return v, fmt.Errorf("execQuery: %w", err)
	}
	result := raw.ToIDispatch()
	defer result.Release()

	itemRaw, err := oleutil.CallMethod(result, "ItemIndex", 0)
	if err != nil {
		v.bitlockerClose()
		return v, fmt.Errorf("failed to fetch result row while processing BitLocker info: %w", err)
	}
	v.handle = itemRaw.ToIDispatch()

	return v, nil
}

// intToPercentage converts an int to a percentage string
func intToPercentage(num int32) string {
	percentage := float64(num) / 10000.0
	return fmt.Sprintf("%.2f%%", percentage)
}

// bitsToDrives converts a bit map to a list of drives
func bitsToDrives(bitMap uint32) (drives []string) {
	availableDrives := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	for i := range availableDrives {
		if bitMap&1 == 1 {
			drives = append(drives, availableDrives[i]+":")
		}
		bitMap >>= 1
	}

	return
}

func getLogicalVolumes() ([]string, error) {
	kernel32, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return nil, fmt.Errorf("failed to load kernel32.dll: %w", err)
	}
	defer func() {
		_ = syscall.FreeLibrary(kernel32)
	}()

	getLogicalDrivesHandle, err := syscall.GetProcAddress(kernel32, "GetLogicalDrives")
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure address: %w", err)
	}

	ret, _, callErr := syscall.SyscallN(getLogicalDrivesHandle, 0, 0, 0, 0)
	if callErr != 0 {
		return nil, fmt.Errorf("syscall to GetLogicalDrives failed: %w", callErr)
	}

	return bitsToDrives(uint32(ret)), nil
}

func getBitlockerStatus(targetVolume string) (*EncryptionStatus, error) {
	// Connect to the volume
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return nil, fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Get volume status
	status, err := vol.getBitlockerStatus()
	if err != nil {
		return nil, fmt.Errorf("starting decryption: %w", err)
	}

	return status, nil
}

/////////////////////////////////////////////////////
// Bitlocker Management interface implementation
/////////////////////////////////////////////////////

// encryptionFlagFromRegistry reads the OSEncryptionType Group Policy registry value to determine
// the encryption flag. Other MDM solutions may set this key to require full disk
// encryption, and the key persists after unenrolling. If the policy requires full disk encryption,
// we honor it; otherwise we default to used-space-only.
//
// Registry values for OSEncryptionType (under HKLM\SOFTWARE\Policies\Microsoft\FVE):
//   - 0: allow user to choose (default to used-space-only)
//   - 1: full disk encryption required
//   - 2: used-space-only encryption
//
// See https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/configure
func encryptionFlagFromRegistry() EncryptionFlag {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\FVE`, registry.QUERY_VALUE)
	if err != nil {
		return EncryptDataOnly
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("OSEncryptionType")
	if err != nil {
		return EncryptDataOnly
	}

	if val == 1 {
		log.Info().Msg("OSEncryptionType registry policy requires full disk encryption, using full encryption mode")
		return EncryptDemandWipe
	}

	return EncryptDataOnly
}

// deleteOSEncryptionTypeRegistry removes the OSEncryptionType value from the FVE policy registry
// key. This cleans up orphaned policy keys left by other MDM solutions after unenrolling.
func deleteOSEncryptionTypeRegistry() {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\FVE`, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	if err := k.DeleteValue("OSEncryptionType"); err != nil {
		log.Debug().Err(err).Msg("could not delete OSEncryptionType registry value")
	} else {
		log.Info().Msg("deleted orphaned OSEncryptionType registry policy value")
	}
}

func encryptVolumeOnCOMThread(targetVolume string) (string, error) {
	// Connect to the volume
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return "", fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Clean up stale key protectors (recovery passwords, TPM, etc.) that may be left over from
	// a previous failed encryption attempt or from another MDM solution. Without this, leftover
	// protectors cause prepareVolume to return ErrorCodeNotDecrypted and subsequent encryption
	// attempts to silently fail. Failures are logged but not fatal since a fresh volume won't
	// have any protectors to delete.
	if err := vol.deleteKeyProtectors(); err != nil {
		log.Debug().Err(err).Msg("could not delete existing key protectors (may not have any), continuing anyway")
	}

	// Read the OSEncryptionType registry policy to determine the encryption flag. If a GPO or
	// another MDM set this to require full disk encryption (value 1), passing the wrong flag
	// to Encrypt() would fail with E_INVALIDARG. We honor the policy to avoid this conflict.
	// If the key is absent or any other value, we default to used-space-only (EncryptDataOnly).
	encFlag := encryptionFlagFromRegistry()

	// Delete the registry key now that we've read it. If it was orphaned from a previous MDM,
	// this cleans it up permanently. In practice, customers should not use GPO for BitLocker
	// policy alongside Fleet MDM since the two will conflict. However, if an active GPO is present,
	// it will re-apply the key on its next refresh (~90 minutes). Orbit retries every ~30
	// seconds on failure, so interim retries before the GPO re-applies the key will default to
	// EncryptDataOnly and fail with E_INVALIDARG. That's expected and the error is reported to
	// Fleet. Once the GPO restores the key, the next retry reads it and succeeds.
	deleteOSEncryptionTypeRegistry()

	// Prepare for encryption
	if err := vol.prepareVolume(VolumeTypeDefault, EncryptionTypeSoftware); err != nil {
		// A previous failed encryption attempt may have already called PrepareVolume, which sets
		// BitLocker metadata on the volume. Calling it again returns ErrorCodeNotDecrypted
		// (FVE_E_NOT_DECRYPTED). This is safe to ignore because the volume is already prepared
		// and we can proceed with adding protectors and encrypting.
		var encErr *EncryptionError
		if !errors.As(err, &encErr) || encErr.Code() != ErrorCodeNotDecrypted {
			return "", fmt.Errorf("preparing volume for encryption: %w", err)
		}
		log.Debug().Msg("volume already prepared from previous attempt, continuing")
	}

	// Add a recovery protector
	recoveryKey, err := vol.protectWithNumericalPassword()
	if err != nil {
		return "", fmt.Errorf("adding a recovery protector: %w", err)
	}

	// Protect with TPM
	if err := vol.protectWithTPM(nil); err != nil {
		return "", fmt.Errorf("protecting with TPM: %w", err)
	}

	// Start encryption using the flag determined from the registry policy
	if err := vol.encrypt(XtsAES256, encFlag); err != nil {
		return "", fmt.Errorf("starting encryption: %w", err)
	}

	return recoveryKey, nil
}

// rotateRecoveryKeyOnCOMThread rotates the recovery key on an already-encrypted volume.
// It adds a new Fleet-managed recovery key protector, removes old recovery key protectors,
// and returns the new recovery key for escrow. The disk is never decrypted.
func rotateRecoveryKeyOnCOMThread(targetVolume string) (string, error) {
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return "", fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Get existing numerical password (recovery key) protector IDs before adding a new one.
	oldProtectorIDs, err := vol.getKeyProtectorIDs(KeyProtectorTypeNumericalPassword)
	if err != nil {
		return "", fmt.Errorf("listing existing recovery key protectors: %w", err)
	}

	// Add a new recovery key protector. Windows generates the recovery password.
	newRecoveryKey, err := vol.protectWithNumericalPassword()
	if err != nil {
		return "", fmt.Errorf("adding new recovery key protector: %w", err)
	}

	// Remove old recovery key protectors so previously compromised keys are invalidated.
	for _, oldID := range oldProtectorIDs {
		if err := vol.deleteKeyProtector(oldID); err != nil {
			log.Warn().Err(err).Str("protector_id", oldID).Msg("could not delete old recovery key protector, continuing")
		}
	}

	// Ensure a TPM protector exists (some pre-encrypted disks may not have one).
	if err := vol.protectWithTPM(nil); err != nil {
		// ErrorCodeProtectorExists is expected if a TPM protector is already present.
		var encErr *EncryptionError
		if !errors.As(err, &encErr) || encErr.Code() != ErrorCodeProtectorExists {
			log.Debug().Err(err).Msg("could not add TPM protector, continuing")
		}
	}

	return newRecoveryKey, nil
}

func getEncryptionStatusOnCOMThread() ([]VolumeStatus, error) {
	drives, err := getLogicalVolumes()
	if err != nil {
		return nil, fmt.Errorf("logical volumen enumeration %w", err)
	}

	// iterate drives
	var volumeStatus []VolumeStatus
	for _, drive := range drives {
		status, err := getBitlockerStatus(drive)
		if err == nil {
			// Skipping errors on purpose
			driveStatus := VolumeStatus{
				DriveVolume: drive,
				Status:      status,
			}
			volumeStatus = append(volumeStatus, driveStatus)
		}
	}

	return volumeStatus, nil
}
