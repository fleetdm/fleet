package bitlocker

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

// Free space wiping status.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
const (
	WipingStatusFreeSpaceNotWiped         int32 = 0
	WipingStatusFreeSpaceWiped            int32 = 1
	WipingStatusFreeSpaceWipingInProgress int32 = 2
	WipingStatusFreeSpaceWipingPaused     int32 = 3
)

// Specifies whether the volume and the encryption key (if any) are secured.
//
// Values and their meanings were taken from:
// https://learn.microsoft.com/en-us/windows/win32/secprov/getprotectionstatus-win32-encryptablevolume
const (
	ProtectionStatusUnprotected int32 = 0
	ProtectionStatusProtected   int32 = 1
	ProtectionStatusUnknown     int32 = 2
)

const (
	// Error Codes
	ErrorCodeIODevice                   int32 = -2147023779
	ErrorCodeDriveIncompatibleVolume    int32 = -2144272206
	ErrorCodeNoTPMWithPassphrase        int32 = -2144272212
	ErrorCodePassphraseTooLong          int32 = -2144272214
	ErrorCodePolicyPassphraseNotAllowed int32 = -2144272278
	ErrorCodeNotDecrypted               int32 = -2144272327
	ErrorCodeInvalidPasswordFormat      int32 = -2144272331
	ErrorCodeBootableCDOrDVD            int32 = -2144272336
	ErrorCodeProtectorExists            int32 = -2144272335
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
