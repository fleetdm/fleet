package pgsql

import (
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func (s *PgSQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE devices SET bootstrap_token_b64 = $1, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = $2;`,
		nullEmptyString(msg.BootstrapToken.BootstrapToken.String()),
		r.ID,
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *PgSQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenB64 string
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT bootstrap_token_b64 FROM devices WHERE id = $1;`,
		r.ID,
	).Scan(&tokenB64)
	if err != nil {
		return nil, err
	}
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(tokenB64)
	if err == nil {
		err = s.updateLastSeen(r)
	}
	return bsToken, err
}
