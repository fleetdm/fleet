package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/glebarez/sqlite"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/util"
)

// Supported DB dialects.
const (
	dialectSqlite3    = "sqlite3"
	dialectMysql      = "mysql"
	dialectPostgreSQL = "postgres"
)

// RDBDriver is Driver for RDB
type RDBDriver struct {
	name string
	conn *gorm.DB
}

// https://github.com/mattn/go-sqlite3/blob/edc3bb69551dcfff02651f083b21f3366ea2f5ab/error.go#L18-L66
type errNo int

type sqliteError struct {
	Code errNo /* The error code returned by SQLite */
}

// result codes from http://www.sqlite.org/c3ref/c_abort.html
var (
	errBusy   = errNo(5) /* The database file is locked */
	errLocked = errNo(6) /* A table in the database is locked */
)

// ErrDBLocked :
var ErrDBLocked = xerrors.New("database is locked")

// Name is driver name
func (r *RDBDriver) Name() string {
	return r.name
}

// OpenDB opens Database
func (r *RDBDriver) OpenDB(dbType, dbPath string, debugSQL bool, _ Option) (err error) {
	gormConfig := gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel: logger.Silent,
			},
		),
	}

	if debugSQL {
		gormConfig.Logger = logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Info,
				Colorful:      true,
			},
		)
	}

	switch r.name {
	case dialectSqlite3:
		r.conn, err = gorm.Open(sqlite.Open(dbPath), &gormConfig)
		if err != nil {
			parsedErr, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				return xerrors.Errorf("Failed to marshal err. err: %w", marshalErr)
			}

			var errMsg sqliteError
			if unmarshalErr := json.Unmarshal(parsedErr, &errMsg); unmarshalErr != nil {
				return xerrors.Errorf("Failed to unmarshal. err: %w", unmarshalErr)
			}

			switch errMsg.Code {
			case errBusy, errLocked:
				return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, ErrDBLocked)
			default:
				return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
			}
		}

		r.conn.Exec("PRAGMA foreign_keys = ON")
	case dialectMysql:
		r.conn, err = gorm.Open(mysql.Open(dbPath), &gormConfig)
		if err != nil {
			return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
		}
	case dialectPostgreSQL:
		r.conn, err = gorm.Open(postgres.Open(dbPath), &gormConfig)
		if err != nil {
			return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
		}
	default:
		return xerrors.Errorf("Not Supported DB dialects. r.name: %s", r.name)
	}
	return nil
}

// MigrateDB migrates Database
func (r *RDBDriver) MigrateDB() error {
	if err := r.conn.AutoMigrate(
		&models.FetchMeta{},
		&models.Root{},
		&models.Definition{},
		&models.Package{},
		&models.Reference{},
		&models.Advisory{},
		&models.Cve{},
		&models.Bugzilla{},
		&models.Resolution{},
		&models.Component{},
		&models.Cpe{},
		&models.Debian{},
	); err != nil {
		switch r.name {
		case dialectSqlite3:
			if r.name == dialectSqlite3 {
				parsedErr, marshalErr := json.Marshal(err)
				if marshalErr != nil {
					return xerrors.Errorf("Failed to marshal err. err: %w", marshalErr)
				}

				var errMsg sqliteError
				if unmarshalErr := json.Unmarshal(parsedErr, &errMsg); unmarshalErr != nil {
					return xerrors.Errorf("Failed to unmarshal. err: %w", unmarshalErr)
				}

				switch errMsg.Code {
				case errBusy, errLocked:
					return xerrors.Errorf("Failed to migrate. err: %w", ErrDBLocked)
				default:
					return xerrors.Errorf("Failed to migrate. err: %w", err)
				}
			}
		case dialectMysql, dialectPostgreSQL:
			return xerrors.Errorf("Failed to migrate. err: %w", err)
		default:
			return xerrors.Errorf("Not Supported DB dialects. r.name: %s", r.name)
		}
	}

	return nil
}

// CloseDB close Database
func (r *RDBDriver) CloseDB() (err error) {
	if r.conn == nil {
		return
	}

	var sqlDB *sql.DB
	if sqlDB, err = r.conn.DB(); err != nil {
		return xerrors.Errorf("Failed to get DB Object. err : %w", err)
	}
	if err = sqlDB.Close(); err != nil {
		return xerrors.Errorf("Failed to close DB. Type: %s. err: %w", r.name, err)
	}
	return
}

// GetByPackName select OVAL definition related to OS Family, osVer, packName
func (r *RDBDriver) GetByPackName(family, osVer, packName, arch string) ([]models.Definition, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	q := r.conn.
		Joins("JOIN roots ON roots.id = definitions.root_id AND roots.family= ? AND roots.os_version = ?", family, osVer).
		Joins("JOIN packages ON packages.definition_id = definitions.id").
		Preload("Advisory").
		Preload("Advisory.Cves").
		Preload("Advisory.Bugzillas").
		Preload("Advisory.AffectedResolution").
		Preload("Advisory.AffectedResolution.Components").
		Preload("Advisory.AffectedCPEList").
		Preload("References")

	switch family {
	case c.Debian:
		q = q.Preload("Debian").Where("packages.name = ?", packName).Preload("AffectedPacks")
	case c.Amazon, c.Oracle, c.Fedora:
		if arch == "" {
			q = q.Where("packages.name = ?", packName).Preload("AffectedPacks")
		} else {
			q = q.Where("packages.name = ? AND packages.arch = ?", packName, arch).Preload("AffectedPacks", "arch = ?", arch)
		}
	default:
		q = q.Where("packages.name = ?", packName).Preload("AffectedPacks")
	}

	defs := []models.Definition{}
	tmpDefs := []models.Definition{}
	if err := q.FindInBatches(&tmpDefs, 998, func(_ *gorm.DB, _ int) error {
		defs = append(defs, tmpDefs...)
		return nil
	}).Error; err != nil {
		return nil, xerrors.Errorf("Failed to FindInBatches. family: %s, osVer: %s, packName: %s, arch: %s, err: %w", family, osVer, packName, arch, err)
	}

	switch family {
	case c.RedHat:
		for i := range defs {
			defs[i].AffectedPacks = filterByRedHatMajor(defs[i].AffectedPacks, major(osVer))
		}
		return defs, nil
	case c.OpenSUSE, c.OpenSUSELeap, c.SUSEEnterpriseDesktop, c.SUSEEnterpriseServer:
		m := map[string]models.Definition{}
		for _, d := range defs {
			m[d.DefinitionID] = d
		}
		return slices.Collect(maps.Values(m)), nil
	default:
		return defs, nil
	}
}

// GetByCveID select OVAL definition related to OS Family, osVer, cveID
func (r *RDBDriver) GetByCveID(family, osVer, cveID, arch string) ([]models.Definition, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	q := r.conn.
		Joins("JOIN roots ON roots.id = definitions.root_id AND roots.family= ? AND roots.os_version = ?", family, osVer).
		Joins("JOIN advisories ON advisories.definition_id = definitions.id").
		Joins("JOIN cves ON cves.advisory_id = advisories.id").
		Where("cves.cve_id = ?", cveID).
		Preload("Advisory").
		Preload("Advisory.Cves").
		Preload("Advisory.Bugzillas").
		Preload("Advisory.AffectedResolution").
		Preload("Advisory.AffectedResolution.Components").
		Preload("Advisory.AffectedCPEList").
		Preload("References")

	switch family {
	case c.Debian:
		q = q.Preload("Debian").Preload("AffectedPacks")
	case c.Amazon, c.Oracle, c.Fedora:
		if arch == "" {
			q = q.Preload("AffectedPacks")
		} else {
			q = q.Preload("AffectedPacks", "arch = ?", arch)
		}
	default:
		q = q.Preload("AffectedPacks")
	}

	defs := []models.Definition{}
	if err := q.Find(&defs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	switch family {
	case c.RedHat:
		for i := range defs {
			defs[i].AffectedPacks = filterByRedHatMajor(defs[i].AffectedPacks, major(osVer))
		}
		return defs, nil
	case c.OpenSUSE, c.OpenSUSELeap, c.SUSEEnterpriseDesktop, c.SUSEEnterpriseServer:
		m := map[string]models.Definition{}
		for _, d := range defs {
			m[d.DefinitionID] = d
		}
		return slices.Collect(maps.Values(m)), nil
	default:
		return defs, nil
	}
}

// GetAdvisories select AdvisoryID: []CVE IDs
func (r *RDBDriver) GetAdvisories(family, osVer string) (map[string][]string, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	q := r.conn.
		Joins("JOIN roots ON roots.id = definitions.root_id AND roots.family= ? AND roots.os_version = ?", family, osVer).
		Joins("JOIN advisories ON advisories.definition_id = definitions.id").
		Joins("JOIN cves ON cves.advisory_id = advisories.id").
		Preload("Advisory").
		Preload("Advisory.Cves")

	switch family {
	case c.Debian:
		q = q.Preload("Debian", "dsa != \"\"")
	case c.Ubuntu:
		q = q.Preload("References", "ref_url LIKE ?", "https://ubuntu.com/security/notices/USN-%")
	case c.RedHat:
		q = q.Not("definitions.definition_id LIKE ?", "oval:com.redhat.cve:def:%").Preload("References", "source = ?", "RHSA")
	case c.Oracle:
		q = q.Preload("References", "source = ?", "elsa")
	case c.Amazon, c.Fedora:
	case c.Alpine:
		return nil, nil
	case c.OpenSUSE, c.OpenSUSELeap, c.SUSEEnterpriseServer, c.SUSEEnterpriseDesktop:
		q = q.Preload("References", "source = ?", "SUSE-SU")
	default:
		return nil, nil
	}

	defs := []models.Definition{}
	if err := q.Find(&defs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	m := map[string][]string{}
	switch family {
	case c.Debian:
		for _, d := range defs {
			if d.Debian == nil || d.Debian.DSA == "" {
				continue
			}
			for _, cve := range d.Advisory.Cves {
				m[d.Debian.DSA] = append(m[d.Debian.DSA], cve.CveID)
			}
		}
		for k := range m {
			m[k] = util.Unique(m[k])
		}
		return m, nil
	case c.Ubuntu:
		for _, d := range defs {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				m[strings.TrimPrefix(r.RefURL, "https://ubuntu.com/security/notices/")] = append(m[strings.TrimPrefix(r.RefURL, "https://ubuntu.com/security/notices/")], cves...)
			}
		}
		for k := range m {
			m[k] = util.Unique(m[k])
		}
		return m, nil
	case c.RedHat, c.Oracle:
		for _, d := range defs {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				m[r.RefID] = append(m[r.RefID], cves...)
			}
		}
		for k := range m {
			m[k] = util.Unique(m[k])
		}
		return m, nil
	case c.Amazon, c.Fedora:
		for _, d := range defs {
			for _, cve := range d.Advisory.Cves {
				m[d.Title] = append(m[d.Title], cve.CveID)
			}
		}
		return m, nil
	case c.OpenSUSE, c.OpenSUSELeap, c.SUSEEnterpriseServer, c.SUSEEnterpriseDesktop:
		for _, d := range defs {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				m[r.RefID] = append(m[r.RefID], cves...)
			}
		}
		for k := range m {
			m[k] = util.Unique(m[k])
		}
		return m, nil
	default:
		return nil, nil
	}
}

// InsertOval inserts OVAL
func (r *RDBDriver) InsertOval(root *models.Root) error {
	family, osVer, err := formatFamilyAndOSVer(root.Family, root.OSVersion)
	if err != nil {
		return xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}
	log15.Info("Refreshing...", "Family", family, "Version", osVer)

	batchSize := viper.GetInt("batch-size")
	if batchSize < 1 {
		return fmt.Errorf("Failed to set batch-size. err: batch-size option is not set properly")
	}

	tx := r.conn.Begin()
	old := models.Root{}
	result := tx.Where(&models.Root{Family: family, OSVersion: osVer}).First(&old)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return xerrors.Errorf("Failed to select old defs: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log15.Info("Deleting old Definitions...")
		// Delete data related to root passed in arg
		defs := []models.Definition{}
		if err := tx.Model(&old).Association("Definitions").Find(&defs); err != nil {
			tx.Rollback()
			return xerrors.Errorf("Failed to select old defs: %w", err)
		}

		bar := pb.StartNew(len(defs)).SetWriter(func() io.Writer {
			if viper.GetBool("log-json") {
				return io.Discard
			}
			return os.Stderr
		}())
		for chunk := range slices.Chunk(defs, 998) {
			var advs []models.Advisory
			if err := tx.Model(chunk).Association("Advisory").Find(&advs); err != nil {
				tx.Rollback()
				return xerrors.Errorf("Failed to delete: %w", err)
			}

			for chunk2 := range slices.Chunk(advs, 998) {
				if err := tx.Select(clause.Associations).Unscoped().Delete(chunk2).Error; err != nil {
					tx.Rollback()
					return xerrors.Errorf("Failed to delete: %w", err)
				}
			}

			if err := tx.Select(clause.Associations).Unscoped().Delete(chunk).Error; err != nil {
				tx.Rollback()
				return xerrors.Errorf("Failed to delete: %w", err)
			}
			bar.Add(len(chunk))
		}
		if err := tx.Unscoped().Where("id = ?", old.ID).Delete(&models.Root{}).Error; err != nil {
			tx.Rollback()
			return xerrors.Errorf("Failed to delete: %w", err)
		}
		bar.Finish()
	}

	log15.Info("Inserting new Definitions...")
	bar := pb.StartNew(len(root.Definitions)).SetWriter(func() io.Writer {
		if viper.GetBool("log-json") {
			return io.Discard
		}
		return os.Stderr
	}())
	if err := tx.Omit("Definitions").Create(&root).Error; err != nil {
		tx.Rollback()
		return xerrors.Errorf("Failed to insert Root. err: %w", err)
	}

	for i := range root.Definitions {
		root.Definitions[i].RootID = root.ID
	}

	for chunk := range slices.Chunk(root.Definitions, batchSize) {
		if err := tx.Omit("Advisory.Cves", "AffectedPacks").Create(chunk).Error; err != nil {
			tx.Rollback()
			return xerrors.Errorf("Failed to insert Definitions. err: %w", err)
		}

		for _, d := range chunk {
			for i := range d.Advisory.Cves {
				d.Advisory.Cves[i].AdvisoryID = d.Advisory.ID
			}
			for chunk2 := range slices.Chunk(d.Advisory.Cves, batchSize) {
				if err := tx.Create(chunk2).Error; err != nil {
					tx.Rollback()
					return xerrors.Errorf("Failed to insert CVEs. err: %w", err)
				}
			}

			for i := range d.AffectedPacks {
				d.AffectedPacks[i].DefinitionID = d.ID
			}
			for chunk2 := range slices.Chunk(d.AffectedPacks, batchSize) {
				if err := tx.Create(chunk2).Error; err != nil {
					tx.Rollback()
					return xerrors.Errorf("Failed to insert AffectedPacks. err: %w", err)
				}
			}
		}

		bar.Add(len(chunk))
	}
	bar.Finish()

	return tx.Commit().Error
}

// CountDefs counts the number of definitions specified by args
func (r *RDBDriver) CountDefs(family, osVer string) (int, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return 0, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	root := models.Root{}
	if err := r.conn.Where(&models.Root{Family: family, OSVersion: osVer}).Take(&root).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
		return 0, nil
	}

	var count int64
	if err := r.conn.Model(&models.Definition{}).Where(
		"root_id = ?", root.ID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetLastModified get last modified time of OVAL in roots
func (r *RDBDriver) GetLastModified(family, osVer string) (time.Time, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return time.Time{}, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	root := models.Root{}
	result := r.conn.Where(&models.Root{Family: family, OSVersion: osVer}).First(&root)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return time.Time{}, xerrors.Errorf("Failed to get root: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		now := time.Now()
		return now.AddDate(-100, 0, 0), nil
	}
	return root.Timestamp, nil
}

// IsGovalDictModelV1 determines if the DB was created at the time of goval-dictionary Model v1
func (r *RDBDriver) IsGovalDictModelV1() (bool, error) {
	return r.conn.Migrator().HasColumn(&models.FetchMeta{}, "file_name"), nil
}

// GetFetchMeta get FetchMeta from Database
func (r *RDBDriver) GetFetchMeta() (fetchMeta *models.FetchMeta, err error) {
	if err = r.conn.Take(&fetchMeta).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return &models.FetchMeta{GovalDictRevision: c.Revision, SchemaVersion: models.LatestSchemaVersion, LastFetchedAt: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)}, nil
	}

	return fetchMeta, nil
}

// UpsertFetchMeta upsert FetchMeta to Database
func (r *RDBDriver) UpsertFetchMeta(fetchMeta *models.FetchMeta) error {
	fetchMeta.GovalDictRevision = c.Revision
	fetchMeta.SchemaVersion = models.LatestSchemaVersion
	return r.conn.Save(fetchMeta).Error
}
