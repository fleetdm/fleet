//go:build !darwin

package useraction

func RotateDiskEncryptionKey(maxRetries int) error {
	return nil
}
