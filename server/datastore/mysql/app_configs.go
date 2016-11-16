package mysql

import (
	"database/sql"

	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	var (
		err    error
		result sql.Result
	)

	err = d.db.Get(info, "SELECT * FROM app_configs LIMIT 1")
	switch err {
	case sql.ErrNoRows:
		result, err = d.db.Exec(
			"INSERT INTO app_configs (org_name, org_logo_url, kolide_server_url) VALUES (?, ?, ?)",
			info.OrgName, info.OrgLogoURL, info.KolideServerURL,
		)
		if err != nil {
			return nil, err
		}

		info.ID, _ = result.LastInsertId()
		return info, nil
	case nil:
		return info, d.SaveAppConfig(info)
	default:
		return nil, err
	}
}

func (d *Datastore) AppConfig() (*kolide.AppConfig, error) {
	info := &kolide.AppConfig{}
	err := d.db.Get(info, "SELECT * FROM app_configs LIMIT 1")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (d *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	_, err := d.db.Exec(
		"UPDATE app_configs SET org_name = ?, org_logo_url = ?, kolide_server_url = ? WHERE id = ?",
		info.OrgName, info.OrgLogoURL, info.KolideServerURL, info.ID,
	)
	return err
}
