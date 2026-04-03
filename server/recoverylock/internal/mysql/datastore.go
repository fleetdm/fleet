// Package mysql implements the MySQL datastore for the recovery lock bounded context.
// This package should only be imported within the recoverylock module.
package mysql

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
	"github.com/jmoiron/sqlx"
)

// statusPending is the MDM delivery status for pending commands.
const statusPending = "pending"

// statusVerified is the MDM delivery status for verified commands.
const statusVerified = "verified"

// statusFailed is the MDM delivery status for failed commands.
const statusFailed = "failed"

// operationInstall is the operation type for installing a recovery lock.
const operationInstall = "install"

// operationRemove is the operation type for removing a recovery lock.
const operationRemove = "remove"

// Datastore implements the internal types.Datastore interface for MySQL.
type Datastore struct {
	reader func(ctx context.Context) sqlx.QueryerContext
	writer func(ctx context.Context) sqlx.ExtContext
	// withRetryTxx executes a function within a transaction with retry logic.
	withRetryTxx func(ctx context.Context, fn func(tx sqlx.ExtContext) error) error
	// serverPrivateKey is used for encrypting/decrypting passwords.
	serverPrivateKey string
}

// ReaderFunc is a function type for getting a read-only database connection.
type ReaderFunc func(ctx context.Context) sqlx.QueryerContext

// WriterFunc is a function type for getting a read-write database connection.
type WriterFunc func(ctx context.Context) sqlx.ExtContext

// TxFunc is a function type for executing within a transaction.
type TxFunc func(ctx context.Context, fn func(tx sqlx.ExtContext) error) error

// New creates a new MySQL datastore for recovery lock operations.
func New(reader ReaderFunc, writer WriterFunc, withRetryTxx TxFunc, serverPrivateKey string) *Datastore {
	return &Datastore{
		reader:           reader,
		writer:           writer,
		withRetryTxx:     withRetryTxx,
		serverPrivateKey: serverPrivateKey,
	}
}

// encrypt encrypts a plaintext password using AES-GCM.
func (ds *Datastore) encrypt(plainText []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(ds.serverPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

// decrypt decrypts an encrypted password using AES-GCM.
func (ds *Datastore) decrypt(encrypted []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(ds.serverPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

// Encrypt encrypts a plaintext password string. This is an exported version
// of encrypt for use by the bootstrap package's PasswordEncryptor.
func (ds *Datastore) Encrypt(plaintext string) ([]byte, error) {
	return ds.encrypt([]byte(plaintext))
}

// Decrypt decrypts an encrypted password and returns it as a string.
// This is an exported version of decrypt for use by the bootstrap package's PasswordEncryptor.
func (ds *Datastore) Decrypt(ciphertext []byte) (string, error) {
	decrypted, err := ds.decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// notFoundError is a sentinel error type for not found errors.
type notFoundError struct {
	resourceType string
	message      string
}

func (e *notFoundError) Error() string {
	if e.message != "" {
		return fmt.Sprintf("%s not found: %s", e.resourceType, e.message)
	}
	return fmt.Sprintf("%s not found", e.resourceType)
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

func notFound(resourceType string) *notFoundError {
	return &notFoundError{resourceType: resourceType}
}

func (e *notFoundError) WithMessage(msg string) *notFoundError {
	e.message = msg
	return e
}

// Verify interface compliance
var _ types.Datastore = (*Datastore)(nil)
