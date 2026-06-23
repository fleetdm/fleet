package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

type orgLogoStore struct {
	ds *Datastore
}

var _ fleet.OrgLogoStore = (*orgLogoStore)(nil)

func (ds *Datastore) NewOrgLogoStore() fleet.OrgLogoStore {
	return &orgLogoStore{ds: ds}
}

func (s *orgLogoStore) Put(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading org logo content")
	}
	const stmt = `
		INSERT INTO org_logo (mode, data, uploaded_at)
		VALUES (?, ?, NOW(6))
		ON DUPLICATE KEY UPDATE data = ?, uploaded_at = NOW(6)`
	if _, err := s.ds.writer(ctx).ExecContext(ctx, stmt, string(mode), data, data); err != nil {
		return ctxerr.Wrap(ctx, err, "storing org logo")
	}
	return nil
}

func (s *orgLogoStore) Get(ctx context.Context, mode fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	var data []byte
	err := sqlx.GetContext(ctx, s.ds.reader(ctx), &data, `SELECT data FROM org_logo WHERE mode = ?`, string(mode))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ctxerr.Wrap(ctx, notFound("OrgLogo").WithName(string(mode)), "get org logo")
		}
		return nil, 0, ctxerr.Wrap(ctx, err, "get org logo")
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (s *orgLogoStore) Delete(ctx context.Context, mode fleet.OrgLogoMode) error {
	if _, err := s.ds.writer(ctx).ExecContext(ctx, `DELETE FROM org_logo WHERE mode = ?`, string(mode)); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting org logo")
	}
	return nil
}

func (s *orgLogoStore) Exists(ctx context.Context, mode fleet.OrgLogoMode) (bool, error) {
	var exists bool
	err := sqlx.GetContext(ctx, s.ds.reader(ctx), &exists, `SELECT EXISTS(SELECT 1 FROM org_logo WHERE mode = ?)`, string(mode))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "checking org logo existence")
	}
	return exists, nil
}
