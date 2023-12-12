package bitlocker

// Encryption Status
type EncryptionStatus struct {
	ProtectionStatusDesc string
	ConversionStatusDesc string
	EncryptionPercentage string
	EncryptionFlags      string
	WipingStatusDesc     string
	WipingPercentage     string
}

// Volume Encryption Status
type VolumeStatus struct {
	DriveVolume string
	Status      *EncryptionStatus
}
