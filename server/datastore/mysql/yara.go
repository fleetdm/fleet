package mysql

import (
	"database/sql"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewYARASignatureGroup(ysg *kolide.YARASignatureGroup, opts ...kolide.OptionalArg) (sg *kolide.YARASignatureGroup, err error) {
	dbTx := d.getTransaction(opts)

	sqlStatement := `
    INSERT INTO yara_signatures (
      signature_name
    ) VALUES( ? )
  `
	var result sql.Result
	result, err = dbTx.Exec(sqlStatement, ysg.SignatureName)
	if isDuplicate(err) {
		return nil, alreadyExists("yara signature", 0)
	}
	if err != nil {
		return nil, errors.Wrap(err, "inserting new yara signature group")
	}
	id, _ := result.LastInsertId()
	ysg.ID = uint(id)
	sqlStatement = `
    INSERT INTO yara_signature_paths (
      file_path,
      yara_signature_id
    ) VALUES( ?, ? )
  `

	for _, path := range ysg.Paths {
		_, err = dbTx.Exec(sqlStatement, path, ysg.ID)
		if err != nil {
			return nil, errors.Wrap(err, "inserting new signature path")
		}
	}
	return ysg, nil
}

func (d *Datastore) NewYARAFilePath(fileSectionName, sigGroupName string, opts ...kolide.OptionalArg) error {
	dbTx := d.getTransaction(opts)

	sqlStatement := `
    INSERT INTO yara_file_paths (
      file_integrity_monitoring_id,
      yara_signature_id
    ) VALUES (
      (
        SELECT fim.id
          FROM file_integrity_monitorings AS fim
          WHERE fim.section_name = ?
          LIMIT 1
      ),
      (
        SELECT ys.id AS ys
          FROM yara_signatures AS ys
          WHERE ys.signature_name = ?
          LIMIT 1
      )
    )
  `
	_, err := dbTx.Exec(sqlStatement, fileSectionName, sigGroupName)
	if isDuplicate(err) {
		return alreadyExists("yara file path", 0)
	}
	if err != nil {
		return errors.Wrap(err, "inserting yara file path")
	}
	return nil
}

func (d *Datastore) YARASection() (*kolide.YARASection, error) {
	result := &kolide.YARASection{
		Signatures: make(map[string][]string),
		FilePaths:  make(map[string][]string),
	}
	sqlStatement := `
    SELECT s.signature_name, p.file_path
      FROM yara_signatures AS s
      INNER JOIN yara_signature_paths AS p
      ON ( s.id = p.yara_signature_id )
  `
	rows, err := d.db.Query(sqlStatement)
	if err != nil {
		return nil, errors.Wrap(err, "selecting yara information")
	}
	for rows.Next() {
		var sigName, sigPath string
		err = rows.Scan(&sigName, &sigPath)
		if err != nil {
			return nil, errors.Wrap(err, "scanning yara information")
		}
		result.Signatures[sigName] = append(result.Signatures[sigName], sigPath)
	}

	sqlStatement = `
    SELECT f.section_name, y.signature_name
    FROM file_integrity_monitorings AS f
    INNER JOIN yara_file_paths AS yfp
      ON (f.id = yfp.file_integrity_monitoring_id)
    INNER JOIN yara_signatures AS y
      ON (y.id = yfp.yara_signature_id )
  `
	rows, err = d.db.Query(sqlStatement)
	if err != nil {
		return nil, errors.Wrap(err, "selecting yara signatures")
	}
	for rows.Next() {
		var sectionName, signatureName string
		err = rows.Scan(&sectionName, &signatureName)
		if err != nil {
			return nil, errors.Wrap(err, "scanning yara signature values")
		}
		result.FilePaths[sectionName] = append(result.FilePaths[sectionName], signatureName)
	}

	return result, nil
}
