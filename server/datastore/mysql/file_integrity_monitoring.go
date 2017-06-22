package mysql

import (
	"database/sql"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewFIMSection(fp *kolide.FIMSection, opts ...kolide.OptionalArg) (result *kolide.FIMSection, err error) {
	db := d.getTransaction(opts)

	sqlStatement := `
    INSERT INTO file_integrity_monitorings (
      section_name,
      description
    ) VALUES( ?, ?)
  `
	var resp sql.Result
	resp, err = db.Exec(sqlStatement, fp.SectionName, fp.Description)
	if isDuplicate(err) {
		return nil, alreadyExists("fim_section", 0)
	}
	if err != nil {
		return nil, errors.Wrap(err, "creating fim section")
	}
	id, _ := resp.LastInsertId()
	fp.ID = uint(id)
	sqlStatement = `
    INSERT INTO file_integrity_monitoring_files (
      file,
      file_integrity_monitoring_id
    ) VALUES( ?, ? )
  `
	for _, fileName := range fp.Paths {
		_, err = db.Exec(sqlStatement, fileName, fp.ID)
		if err != nil {
			return nil, errors.Wrap(err, "adding path to fim section")
		}
	}
	return fp, nil
}

func (d *Datastore) FIMSections() (kolide.FIMSections, error) {
	sqlStatement := `
    SELECT fim.section_name, mf.file FROM
     file_integrity_monitorings AS fim
     INNER JOIN file_integrity_monitoring_files AS mf
     ON (fim.id = mf.file_integrity_monitoring_id)
  `
	rows, err := d.db.Query(sqlStatement)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("FilePath")
		}
		return nil, errors.Wrap(err, "retrieving fim sections")
	}
	result := make(kolide.FIMSections)
	for rows.Next() {
		var sectionName, fileName string
		err = rows.Scan(&sectionName, &fileName)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving path for fim section")
		}
		result[sectionName] = append(result[sectionName], fileName)
	}
	return result, nil
}
