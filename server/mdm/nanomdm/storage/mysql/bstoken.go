package mysql

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE nano_devices SET bootstrap_token_b64 = ?, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ? LIMIT 1;`,
		nullEmptyString(msg.BootstrapToken.BootstrapToken.String()),
		r.ID,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenB64 sql.NullString
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT bootstrap_token_b64 FROM nano_devices WHERE id = ?;`,
		r.ID,
	).Scan(&tokenB64)
	if err != nil || !tokenB64.Valid {
		return nil, err
	}
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(tokenB64.String)
	if err == nil {
		err = s.updateLastSeen(r)
	}
	return bsToken, err
}
