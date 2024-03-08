//go:build !windows

package bitlocker

func GetRecoveryKeys(targetVolume string) (map[string]string, error) {
	return nil, nil
}

func EncryptVolume(targetVolume string) (string, error) {
	return "", nil
}

func DecryptVolume(targetVolume string) error {
	return nil
}

func GetEncryptionStatus() ([]VolumeStatus, error) {
	return nil, nil
}
