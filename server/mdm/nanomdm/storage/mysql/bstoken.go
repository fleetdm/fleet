package mysql

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	tokenStr := msg.BootstrapToken.BootstrapToken.String()
	if tokenStr == "" {
		// Store NULL for empty tokens.
		_, err := s.db.ExecContext(
			r.Context,
			`UPDATE nano_devices SET bootstrap_token_b64 = NULL, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ? LIMIT 1;`,
			r.ID,
		)
		if err != nil {
			return err
		}
		return s.updateLastSeen(r)
	}

	var value any
	if s.serverPrivateKey != "" {
		encrypted, err := encrypt([]byte(tokenStr), s.serverPrivateKey)
		if err != nil {
			return fmt.Errorf("encrypt bootstrap token: %w", err)
		}
		value = encrypted
	} else {
		value = tokenStr
	}

	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE nano_devices SET bootstrap_token_b64 = ?, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ? LIMIT 1;`,
		value,
		r.ID,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenRaw sql.RawBytes
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT bootstrap_token_b64 FROM nano_devices WHERE id = ?;`,
		r.ID,
	).Scan(&tokenRaw)
	if err != nil {
		return nil, err
	}
	if tokenRaw == nil {
		return nil, nil
	}
	tokenBytes := decryptOrPassthrough(tokenRaw, s.serverPrivateKey)
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(string(tokenBytes))
	if err == nil {
		err = s.updateLastSeen(r)
	}
	return bsToken, err
}
