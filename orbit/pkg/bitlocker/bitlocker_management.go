package bitlocker

// Volume encryption/decryption status.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
const (
	CONVERSION_STATUS_FULLY_DECRYPTED        int32 = 0
	CONVERSION_STATUS_FULLY_ENCRYPTED        int32 = 1
	CONVERSION_STATUS_ENCRYPTION_IN_PROGRESS int32 = 2
	CONVERSION_STATUS_DECRYPTION_IN_PROGRESS int32 = 3
	CONVERSION_STATUS_ENCRYPTION_PAUSED      int32 = 4
	CONVERSION_STATUS_DECRYPTION_PAUSED      int32 = 5
)

// Free space wiping status.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
const (
	WIPING_STATUS_FREE_SPACE_NOT_WIPED          int32 = 0
	WIPING_STATUS_FREE_SPACE_WIPED              int32 = 1
	WIPING_STATUS_FREE_SPACE_WIPING_IN_PROGRESS int32 = 1
	WIPING_STATUS_FREE_SPACE_WIPING_PAUSED      int32 = 1
)

// Specifies whether the volume and the encryption key (if any) are secured.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getprotectionstatus-win32-encryptablevolume
const (
	PROTECTION_STATUS_UNPROTECTED int32 = 0
	PROTECTION_STATUS_PROTECTED   int32 = 1
	PROTECTION_STATUS_UNKNOWN     int32 = 2
)

const (
	// Error Codes
	ERROR_IO_DEVICE                     int32 = -2147023779
	FVE_E_EDRIVE_INCOMPATIBLE_VOLUME    int32 = -2144272206
	FVE_E_NO_TPM_WITH_PASSPHRASE        int32 = -2144272212
	FVE_E_PASSPHRASE_TOO_LONG           int32 = -2144272214
	FVE_E_POLICY_PASSPHRASE_NOT_ALLOWED int32 = -2144272278
	FVE_E_NOT_DECRYPTED                 int32 = -2144272327
	FVE_E_INVALID_PASSWORD_FORMAT       int32 = -2144272331
	FVE_E_BOOTABLE_CDDVD                int32 = -2144272336
	FVE_E_PROTECTOR_EXISTS              int32 = -2144272335
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
}
