//go:build windows

package bitlocker

import (
	"fmt"
	"syscall"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/scjalliance/comshim"
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
	EncryptDataOnly    EncryptionFlag = 0x00000001
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

func encryptErrHandler(val int32) error {
	var msg string

	switch val {
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
		msg = fmt.Sprintf("error code returned during encryption: %d", val)
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

	comshim.Done()
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

// decrypt encrypts the volume
// Example: vol.decrypt()
// https://learn.microsoft.com/en-us/windows/win32/secprov/decrypt-win32-encryptablevolume
func (v *Volume) decrypt() error {
	resultRaw, err := oleutil.CallMethod(v.handle, "Decrypt")
	if err != nil {
		return fmt.Errorf("decrypt(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return fmt.Errorf("decrypt(%s): %w", v.letter, encryptErrHandler(val))
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
	ole.VariantInit(&volumeKeyProtectorID)
	var resultRaw *ole.VARIANT
	var err error

	resultRaw, err = oleutil.CallMethod(v.handle, "ProtectKeyWithNumericalPassword", nil, nil, &volumeKeyProtectorID)
	if err != nil {
		return "", fmt.Errorf("ProtectKeyWithNumericalPassword(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return "", fmt.Errorf("ProtectKeyWithNumericalPassword(%s): %w", v.letter, encryptErrHandler(val))
	}

	var recoveryKey ole.VARIANT
	ole.VariantInit(&recoveryKey)
	resultRaw, err = oleutil.CallMethod(v.handle, "GetKeyProtectorNumericalPassword", volumeKeyProtectorID.ToString(), &recoveryKey)

	if err != nil {
		return "", fmt.Errorf("GetKeyProtectorNumericalPassword(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return "", fmt.Errorf("GetKeyProtectorNumericalPassword(%s): %w", v.letter, encryptErrHandler(val))
	}

	return recoveryKey.ToString(), nil
}

// protectWithPassphrase adds a passphrase key protector
// https://docs.microsoft.com/en-us/windows/win32/secprov/protectkeywithpassphrase-win32-encryptablevolume
func (v *Volume) protectWithPassphrase(passphrase string) (string, error) {
	var volumeKeyProtectorID ole.VARIANT
	ole.VariantInit(&volumeKeyProtectorID)

	resultRaw, err := oleutil.CallMethod(v.handle, "ProtectKeyWithPassphrase", nil, passphrase, &volumeKeyProtectorID)
	if err != nil {
		return "", fmt.Errorf("protectWithPassphrase(%s): %w", v.letter, err)
	} else if val, ok := resultRaw.Value().(int32); val != 0 || !ok {
		return "", fmt.Errorf("protectWithPassphrase(%s): %w", v.letter, encryptErrHandler(val))
	}

	return volumeKeyProtectorID.ToString(), nil
}

// protectWithTPM adds the TPM key protector
// https://docs.microsoft.com/en-us/windows/win32/secprov/protectkeywithtpm-win32-encryptablevolume
func (v *Volume) protectWithTPM(platformValidationProfile *[]uint8) error {
	var volumeKeyProtectorID ole.VARIANT
	ole.VariantInit(&volumeKeyProtectorID)
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

// getProtectorsKeys returns the recovery keys for the volume
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectornumericalpassword-win32-encryptablevolume
func (v *Volume) getProtectorsKeys() (map[string]string, error) {
	keys, err := getKeyProtectors(v.handle)
	if err != nil {
		return nil, fmt.Errorf("getKeyProtectors: %w", err)
	}

	recoveryKeys := make(map[string]string)
	for _, k := range keys {
		var recoveryKey ole.VARIANT
		ole.VariantInit(&recoveryKey)
		recoveryKeyResultRaw, err := oleutil.CallMethod(v.handle, "GetKeyProtectorNumericalPassword", k, &recoveryKey)
		if err != nil {
			continue // No recovery key for this protector
		} else if val, ok := recoveryKeyResultRaw.Value().(int32); val != 0 || !ok {
			continue // No recovery key for this protector
		}
		recoveryKeys[k] = recoveryKey.ToString()
	}

	return recoveryKeys, nil
}

/////////////////////////////////////////////////////
// Helper functions
/////////////////////////////////////////////////////

// bitlockerConnect connects to an encryptable volume in order to manage it.
func bitlockerConnect(driveLetter string) (Volume, error) {
	comshim.Add(1)
	v := Volume{letter: driveLetter}

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		comshim.Done()
		return v, fmt.Errorf("createObject: %w", err)
	}
	defer unknown.Release()

	v.wmiIntf, err = unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		comshim.Done()
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

// getKeyProtectors returns the key protectors for the volume
// https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectors-win32-encryptablevolume
func getKeyProtectors(item *ole.IDispatch) ([]string, error) {
	kp := []string{}
	var keyProtectorResults ole.VARIANT
	ole.VariantInit(&keyProtectorResults)

	keyIDResultRaw, err := oleutil.CallMethod(item, "GetKeyProtectors", 3, &keyProtectorResults)
	if err != nil {
		return nil, fmt.Errorf("unable to get Key Protectors while getting BitLocker info. %s", err.Error())
	} else if val, ok := keyIDResultRaw.Value().(int32); val != 0 || !ok {
		return nil, fmt.Errorf("unable to get Key Protectors while getting BitLocker info. Return code %d", val)
	}

	keyProtectorValues := keyProtectorResults.ToArray().ToValueArray()
	for _, keyIDItemRaw := range keyProtectorValues {
		keyIDItem, ok := keyIDItemRaw.(string)
		if !ok {
			return nil, fmt.Errorf("keyProtectorID wasn't a string")
		}
		kp = append(kp, keyIDItem)
	}

	return kp, nil
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
	defer syscall.FreeLibrary(kernel32)

	getLogicalDrivesHandle, err := syscall.GetProcAddress(kernel32, "GetLogicalDrives")
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure address: %w", err)
	}

	ret, _, callErr := syscall.SyscallN(uintptr(getLogicalDrivesHandle), 0, 0, 0, 0)
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

func GetRecoveryKeys(targetVolume string) (map[string]string, error) {
	// Connect to the volume
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return nil, fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Get recovery keys
	keys, err := vol.getProtectorsKeys()
	if err != nil {
		return nil, fmt.Errorf("retreving protection keys: %w", err)
	}

	return keys, nil
}

func EncryptVolume(targetVolume string) (string, error) {
	// Connect to the volume
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return "", fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Prepare for encryption
	if err := vol.prepareVolume(VolumeTypeDefault, EncryptionTypeSoftware); err != nil {
		return "", fmt.Errorf("preparing volume for encryption: %w", err)
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

	// Start encryption
	if err := vol.encrypt(XtsAES256, EncryptDataOnly); err != nil {
		return "", fmt.Errorf("starting encryption: %w", err)
	}

	return recoveryKey, nil
}

func DecryptVolume(targetVolume string) error {
	// Connect to the volume
	vol, err := bitlockerConnect(targetVolume)
	if err != nil {
		return fmt.Errorf("connecting to the volume: %w", err)
	}
	defer vol.bitlockerClose()

	// Start decryption
	if err := vol.decrypt(); err != nil {
		return fmt.Errorf("starting decryption: %w", err)
	}

	return nil
}

func GetEncryptionStatus() ([]VolumeStatus, error) {
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
