// Package bootstrap provides dependency injection for the recovery lock bounded context.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/recoverylock"
	"github.com/fleetdm/fleet/v4/server/recoverylock/api"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/service"
)

// New creates a new recovery lock service with all dependencies wired up.
func New(
	reader mysql.ReaderFunc,
	writer mysql.WriterFunc,
	withRetryTxx mysql.TxFunc,
	serverPrivateKey string,
	providers recoverylock.DataProviders,
) api.Service {
	// Create the internal datastore
	ds := mysql.New(reader, writer, withRetryTxx, serverPrivateKey)

	// Create the password encryptor using the datastore's encrypt/decrypt
	encryptor := &datastoreEncryptor{
		reader:           reader,
		serverPrivateKey: serverPrivateKey,
	}

	// Create and return the service
	return service.New(ds, providers, encryptor)
}

// datastoreEncryptor implements recoverylock.PasswordEncryptor using the same
// encryption as the MySQL datastore.
type datastoreEncryptor struct {
	reader           mysql.ReaderFunc
	serverPrivateKey string
}

// Encrypt encrypts a plaintext password.
func (e *datastoreEncryptor) Encrypt(plaintext string) ([]byte, error) {
	// Create a temporary datastore just for encryption
	// This is a bit awkward but keeps the encryption logic in one place
	ds := mysql.New(e.reader, nil, nil, e.serverPrivateKey)
	return ds.Encrypt(plaintext)
}

// Decrypt decrypts an encrypted password.
func (e *datastoreEncryptor) Decrypt(ciphertext []byte) (string, error) {
	ds := mysql.New(e.reader, nil, nil, e.serverPrivateKey)
	return ds.Decrypt(ciphertext)
}
