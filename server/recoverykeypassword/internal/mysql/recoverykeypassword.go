// Package mysql provides the MySQL datastore implementation for recovery key passwords.
package mysql

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the recovery key password datastore.
type Datastore struct {
	primary          *sqlx.DB
	replica          *sqlx.DB
	serverPrivateKey string
	logger           *slog.Logger
}

// NewDatastore creates a new MySQL datastore for recovery key passwords.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{
		primary:          conns.Primary,
		replica:          conns.Replica,
		serverPrivateKey: conns.Options.PrivateKey,
		logger:           logger,
	}
}

// Ensure Datastore implements the interface
var _ recoverykeypassword.Datastore = (*Datastore)(nil)

// SetHostRecoveryKeyPassword generates a new recovery key password,
// encrypts it, and stores it for the given host.
func (ds *Datastore) SetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (string, error) {
	password, err := recoverykeypassword.GeneratePassword()
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generating recovery key password")
	}

	encrypted, err := encrypt([]byte(password), ds.serverPrivateKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "encrypting recovery key password")
	}

	const stmt = `
		INSERT INTO host_recovery_key_passwords (host_id, encrypted_password)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password)
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, hostID, encrypted); err != nil {
		return "", ctxerr.Wrap(ctx, err, "storing recovery key password")
	}

	return password, nil
}

// GetHostRecoveryKeyPassword retrieves and decrypts the recovery key password.
func (ds *Datastore) GetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (*recoverykeypassword.HostRecoveryKeyPassword, error) {
	const stmt = `SELECT encrypted_password, updated_at FROM host_recovery_key_passwords WHERE host_id = ?`

	var row struct {
		EncryptedPassword []byte    `db:"encrypted_password"`
		UpdatedAt         time.Time `db:"updated_at"`
	}
	if err := sqlx.GetContext(ctx, ds.replica, &row, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, platform_mysql.NotFound("HostRecoveryKeyPassword").
				WithMessage(fmt.Sprintf("for host %d", hostID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery key password")
	}

	decrypted, err := decrypt(row.EncryptedPassword, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting recovery key password")
	}

	return &recoverykeypassword.HostRecoveryKeyPassword{
		Password:  string(decrypted),
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func encrypt(plainText []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
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

func decrypt(encrypted []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
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
