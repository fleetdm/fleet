package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/go-redis/redis/v8"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/util"
)

/**
# Redis Data Structure

- Strings
  ┌───┬──────────────────────────────────────┬─────────┬──────────────────────────────────────────────────┐
  │NO │                   KEY                │  VALUE  │                   PURPOSE                        │
  └───┴──────────────────────────────────────┴─────────┴──────────────────────────────────────────────────┘
  ┌───┬──────────────────────────────────────┬─────────┬──────────────────────────────────────────────────┐
  │ 1 │ OVAL#$OSFAMILY#$VERSION#DEP          │   JSON  │ TO DELETE OUTDATED AND UNNEEDED FIELD AND MEMBER │
  ├───┼──────────────────────────────────────┼─────────┼──────────────────────────────────────────────────┤
  │ 2 │ OVAL#$OSFAMILY#$VERSION#LASTMODIFIED │  string │ TO GET Last Modified                             │
  └───┴──────────────────────────────────────┴─────────┴──────────────────────────────────────────────────┘

- Sets
  ┌───┬────────────────────────────────────────────────┬───────────────┬──────────────────────────────────────────┐
  │NO │ KEY                                            │     MEMBER    │ PURPOSE                                  │
  └───┴────────────────────────────────────────────────┴───────────────┴──────────────────────────────────────────┘
  ┌───┬────────────────────────────────────────────────┬───────────────┬─────────────────────────────────────────────────┐
  │ 1 │ OVAL#$OSFAMILY#$VERSION#PKG#$PACKAGENAME       │ $DEFINITIONID │ TO GET []$DEFINITIONID                          │
  ├───┼────────────────────────────────────────────────┼───────────────┼─────────────────────────────────────────────────┤
  │ 2 │ OVAL#$OSFAMILY#$VERSION#CVE#$CVEID             │ $DEFINITIONID │ TO GET []$DEFINITIONID                          │
  └───┴────────────────────────────────────────────────┴───────────────┴─────────────────────────────────────────────────┘

- Hash
  ┌───┬─────────────────────────────┬───────────────┬───────────┬───────────────────────────────────────────┐
  │NO │               KEY           │     FIELD     │   VALUE   │                PURPOSE                    │
  └───┴─────────────────────────────┴───────────────┴───────────┴───────────────────────────────────────────┘
  ┌───┬─────────────────────────────┬───────────────┬───────────┬───────────────────────────────────────────┐
  │ 1 │ OVAL#$OSFAMILY#$VERSION#DEF │ $DEFINITIONID │ $OVALJSON │ TO GET OVALJSON                           │
  ├───┼─────────────────────────────┼───────────────┼───────────┼───────────────────────────────────────────┤
  │ 2 │ OVAL#FETCHMETA              │   Revision    │   string  │ GET Go-Oval-Dictionary Binary Revision    │
  ├───┼─────────────────────────────┼───────────────┼───────────┼───────────────────────────────────────────┤
  │ 3 │ OVAL#FETCHMETA              │ SchemaVersion │    uint   │ GET Go-Oval-Dictionary Schema Version     │
  ├───┼─────────────────────────────┼───────────────┼───────────┼───────────────────────────────────────────┤
  │ 4 │ OVAL#FETCHMETA              │ LastFetchedAt │ time.Time │ GET Go-Oval-Dictionary Last Fetched Time  │
  ├───┼─────────────────────────────┼───────────────┼───────────┼───────────────────────────────────────────┤
  │ 5 │ OVAL#$OSFAMILY#$VERSION#ADV │ $ADVISORYID   │ []$CVEID  │ GET CVEIDs related to Advisory ID         │
  └───┴─────────────────────────────┴───────────────┴───────────┴───────────────────────────────────────────┘

  **/

// Supported DB dialects.
const (
	dialectRedis          = "redis"
	defKeyFormat          = "OVAL#%s#%s#DEF"
	cveKeyFormat          = "OVAL#%s#%s#CVE#%s"
	pkgKeyFormat          = "OVAL#%s#%s#PKG#%s"
	advKeyFormat          = "OVAL#%s#%s#ADV"
	depKeyFormat          = "OVAL#%s#%s#DEP"
	lastModifiedKeyFormat = "OVAL#%s#%s#LASTMODIFIED"
	fetchMetaKey          = "OVAL#FETCHMETA"
)

// RedisDriver is Driver for Redis
type RedisDriver struct {
	name string
	conn *redis.Client
}

// Name is driver name
func (r *RedisDriver) Name() string {
	return r.name
}

// OpenDB opens Database
func (r *RedisDriver) OpenDB(_, dbPath string, _ bool, option Option) error {
	if err := r.connectRedis(dbPath, option); err != nil {
		return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dialectRedis, dbPath, err)
	}
	return nil
}

func (r *RedisDriver) connectRedis(dbPath string, option Option) error {
	opt, err := redis.ParseURL(dbPath)
	if err != nil {
		return xerrors.Errorf("Failed to parse url. err: %w", err)
	}

	if 0 < option.RedisTimeout.Seconds() {
		opt.ReadTimeout = option.RedisTimeout
	}
	r.conn = redis.NewClient(opt)
	return r.conn.Ping(context.Background()).Err()
}

// CloseDB close Database
func (r *RedisDriver) CloseDB() (err error) {
	if r.conn == nil {
		return
	}
	if err = r.conn.Close(); err != nil {
		return xerrors.Errorf("Failed to close DB. Type: %s. err: %w", r.name, err)
	}
	return
}

// MigrateDB migrates Database
func (r *RedisDriver) MigrateDB() error {
	return nil
}

// GetByPackName select OVAL definition related to OS Family, osVer, packName, arch
func (r *RedisDriver) GetByPackName(family, osVer, packName, arch string) ([]models.Definition, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	ctx := context.Background()

	defIDs, err := r.conn.SMembers(ctx, fmt.Sprintf(pkgKeyFormat, family, osVer, packName)).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to SMembers. err: %w", err)
	}
	if len(defIDs) == 0 {
		return []models.Definition{}, nil
	}

	defStrs, err := r.conn.HMGet(ctx, fmt.Sprintf(defKeyFormat, family, osVer), defIDs...).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HMGet. err: %w", err)
	}

	defs := []models.Definition{}
	for i, defstr := range defStrs {
		if defstr == nil {
			return nil, xerrors.Errorf("Failed to HMGet. Redis relationship may be broken. err: Some fields do not exist. family: %s, version: %s, defID: %s", family, osVer, defIDs[i])
		}
		def, err := restoreDefinition(defstr.(string), family, osVer, arch)
		if err != nil {
			return nil, xerrors.Errorf("Failed to restoreDefinition. err: %w", err)
		}
		if len(def.AffectedPacks) > 0 {
			defs = append(defs, def)
		}
	}
	return defs, nil
}

// GetByCveID select OVAL definition related to OS Family, osVer, cveID
func (r *RedisDriver) GetByCveID(family, osVer, cveID, arch string) ([]models.Definition, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	ctx := context.Background()
	defIDs, err := r.conn.SMembers(ctx, fmt.Sprintf(cveKeyFormat, family, osVer, cveID)).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to SMembers. err: %w", err)
	}
	if len(defIDs) == 0 {
		return []models.Definition{}, nil
	}

	defStrs, err := r.conn.HMGet(ctx, fmt.Sprintf(defKeyFormat, family, osVer), defIDs...).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HMGet. err: %w", err)
	}

	defs := []models.Definition{}
	for i, defstr := range defStrs {
		if defstr == nil {
			return nil, xerrors.Errorf("Failed to HMGet. Redis relationship may be broken. err: Some fields do not exist. family: %s, version: %s, defID: %s", family, osVer, defIDs[i])
		}
		def, err := restoreDefinition(defstr.(string), family, osVer, arch)
		if err != nil {
			return nil, xerrors.Errorf("Failed to restoreDefinition. err: %w", err)
		}
		if len(def.AffectedPacks) > 0 {
			defs = append(defs, def)
		}
	}
	return defs, nil
}

func restoreDefinition(defstr, family, version, arch string) (models.Definition, error) {
	var def models.Definition
	if err := json.Unmarshal([]byte(defstr), &def); err != nil {
		return models.Definition{}, xerrors.Errorf("Failed to Unmarshal JSON. err: %w", err)
	}

	switch family {
	case c.Amazon, c.Oracle, c.Fedora:
		def.AffectedPacks = fileterPacksByArch(def.AffectedPacks, arch)
	case c.RedHat:
		def.AffectedPacks = filterByRedHatMajor(def.AffectedPacks, major(version))
	}

	return def, nil
}

func fileterPacksByArch(packs []models.Package, arch string) []models.Package {
	if arch == "" {
		return packs
	}

	filtered := []models.Package{}
	for _, pack := range packs {
		if pack.Arch == arch {
			filtered = append(filtered, pack)
		}
	}

	return filtered
}

// GetAdvisories select AdvisoryID: []CVE IDs
func (r *RedisDriver) GetAdvisories(family, osVer string) (map[string][]string, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return nil, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	ctx := context.Background()
	v, err := r.conn.HGetAll(ctx, fmt.Sprintf(advKeyFormat, family, osVer)).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGetAll. err: %w", err)
	}

	m := map[string][]string{}
	for adv, s := range v {
		var cves []string
		if err := json.Unmarshal([]byte(s), &cves); err != nil {
			return nil, xerrors.Errorf("Failed to Unmarshal JSON. err: %w", err)
		}
		m[adv] = cves
	}

	return m, nil
}

// InsertOval inserts OVAL
func (r *RedisDriver) InsertOval(root *models.Root) (err error) {
	ctx := context.Background()
	batchSize := viper.GetInt("batch-size")
	if batchSize < 1 {
		return fmt.Errorf("Failed to set batch-size. err: batch-size option is not set properly")
	}

	family, osVer, err := formatFamilyAndOSVer(root.Family, root.OSVersion)
	if err != nil {
		return xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}
	log15.Info("Refreshing...", "Family", family, "Version", osVer)

	advs := map[string][]string{}
	switch family {
	case c.Debian:
		for _, d := range root.Definitions {
			if d.Debian == nil || d.Debian.DSA == "" {
				continue
			}
			for _, cve := range d.Advisory.Cves {
				advs[d.Debian.DSA] = append(advs[d.Debian.DSA], cve.CveID)
			}
		}
		for k := range advs {
			advs[k] = util.Unique(advs[k])
		}
	case c.Ubuntu:
		for _, d := range root.Definitions {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				if strings.HasPrefix(r.RefURL, "https://ubuntu.com/security/notices/USN-") {
					advs[strings.TrimPrefix(r.RefURL, "https://ubuntu.com/security/notices/")] = append(advs[strings.TrimPrefix(r.RefURL, "https://ubuntu.com/security/notices/")], cves...)
				}
			}
		}
		for k := range advs {
			advs[k] = util.Unique(advs[k])
		}
	case c.RedHat:
		for _, d := range root.Definitions {
			if strings.HasPrefix(d.DefinitionID, "oval:com.redhat.cve:def:") {
				continue
			}
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				if r.Source == "RHSA" {
					advs[r.RefID] = append(advs[r.RefID], cves...)
				}
			}
		}
	case c.Oracle:
		for _, d := range root.Definitions {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				if r.Source == "elsa" {
					advs[r.RefID] = append(advs[r.RefID], cves...)
				}
			}
		}
	case c.Amazon, c.Fedora:
		for _, d := range root.Definitions {
			for _, cve := range d.Advisory.Cves {
				advs[d.Title] = append(advs[d.Title], cve.CveID)
			}
		}
	case c.Alpine:
	case c.OpenSUSE, c.OpenSUSELeap, c.SUSEEnterpriseServer, c.SUSEEnterpriseDesktop:
		for _, d := range root.Definitions {
			cves := make([]string, 0, len(d.Advisory.Cves))
			for _, cve := range d.Advisory.Cves {
				cves = append(cves, cve.CveID)
			}
			for _, r := range d.References {
				if r.Source == "SUSE-SU" {
					advs[r.RefID] = append(advs[r.RefID], cves...)
				}
			}
		}
		for k := range advs {
			advs[k] = util.Unique(advs[k])
		}
	default:
	}

	// newDeps, oldDeps: {"DEFID": {"cves": {"CVEID": {}}, "packages": {"PACKNAME": {}}}, "advisories": {"ADVISORYID": {}}}
	newDeps := map[string]map[string]map[string]struct{}{"advisories": {}}
	depKey := fmt.Sprintf(depKeyFormat, family, osVer)
	oldDepsStr, err := r.conn.Get(ctx, depKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return xerrors.Errorf("Failed to Get key: %s. err: %w", depKey, err)
		}
		oldDepsStr = "{}"
	}
	var oldDeps map[string]map[string]map[string]struct{}
	if err := json.Unmarshal([]byte(oldDepsStr), &oldDeps); err != nil {
		return xerrors.Errorf("Failed to unmarshal JSON. err: %w", err)
	}

	log15.Info("Insert Definitions...")
	bar := pb.StartNew(len(root.Definitions)).SetWriter(func() io.Writer {
		if viper.GetBool("log-json") {
			return io.Discard
		}
		return os.Stderr
	}())
	for chunk := range slices.Chunk(root.Definitions, batchSize) {
		pipe := r.conn.Pipeline()
		for _, def := range chunk {
			var dj []byte
			if dj, err = json.Marshal(def); err != nil {
				return xerrors.Errorf("Failed to marshal json. err: %w", err)
			}

			_ = pipe.HSet(ctx, fmt.Sprintf(defKeyFormat, family, osVer), def.DefinitionID, string(dj))
			if _, ok := newDeps[def.DefinitionID]; !ok {
				newDeps[def.DefinitionID] = map[string]map[string]struct{}{"cves": {}, "packages": {}}
			}

			for _, cve := range def.Advisory.Cves {
				_ = pipe.SAdd(ctx, fmt.Sprintf(cveKeyFormat, family, osVer, cve.CveID), def.DefinitionID)
				newDeps[def.DefinitionID]["cves"][cve.CveID] = struct{}{}
				if _, ok := oldDeps[def.DefinitionID]; ok {
					if _, ok := oldDeps[def.DefinitionID]["cves"]; ok {
						delete(oldDeps[def.DefinitionID]["cves"], cve.CveID)
					}
				}
			}

			for _, pack := range def.AffectedPacks {
				_ = pipe.SAdd(ctx, fmt.Sprintf(pkgKeyFormat, family, osVer, pack.Name), def.DefinitionID)
				newDeps[def.DefinitionID]["packages"][pack.Name] = struct{}{}
				if _, ok := oldDeps[def.DefinitionID]; ok {
					if _, ok := oldDeps[def.DefinitionID]["packages"]; ok {
						delete(oldDeps[def.DefinitionID]["packages"], pack.Name)
					}
				}
			}

			if _, ok := oldDeps[def.DefinitionID]; ok {
				if _, ok := oldDeps[def.DefinitionID]["cves"]; ok {
					if len(oldDeps[def.DefinitionID]["cves"]) == 0 {
						delete(oldDeps[def.DefinitionID], "cves")
					}
				}
				if _, ok := oldDeps[def.DefinitionID]["packages"]; ok {
					if len(oldDeps[def.DefinitionID]["packages"]) == 0 {
						delete(oldDeps[def.DefinitionID], "packages")
					}
				}
				if len(oldDeps[def.DefinitionID]) == 0 {
					delete(oldDeps, def.DefinitionID)
				}
			}
		}
		if _, err = pipe.Exec(ctx); err != nil {
			return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
		}
		bar.Add(len(chunk))
	}
	bar.Finish()

	log15.Info("Insert Advisories...")
	bar = pb.StartNew(len(advs)).SetWriter(func() io.Writer {
		if viper.GetBool("log-json") {
			return io.Discard
		}
		return os.Stderr
	}())

	keys := slices.Collect(maps.Keys(advs))
	for chunk := range slices.Chunk(keys, batchSize) {
		pipe := r.conn.Pipeline()
		for _, adv := range chunk {
			var aj []byte
			if aj, err = json.Marshal(advs[adv]); err != nil {
				return xerrors.Errorf("Failed to marshal json. err: %w", err)
			}

			_ = pipe.HSet(ctx, fmt.Sprintf(advKeyFormat, family, osVer), adv, string(aj))
			newDeps["advisories"][adv] = map[string]struct{}{}
			if _, ok := oldDeps["advisories"]; ok {
				delete(oldDeps["advisories"], adv)
			}
		}
		if _, err = pipe.Exec(ctx); err != nil {
			return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
		}
		bar.Add(len(chunk))
	}
	bar.Finish()

	pipe := r.conn.Pipeline()
	for k, v := range oldDeps {
		switch k {
		case "advisories":
			for advID := range v {
				_ = pipe.HDel(ctx, fmt.Sprintf(advKeyFormat, family, osVer), advID)
			}
		default:
			for cveID := range v["cves"] {
				_ = pipe.SRem(ctx, fmt.Sprintf(cveKeyFormat, family, osVer, cveID), k)
			}
			for pack := range v["packages"] {
				_ = pipe.SRem(ctx, fmt.Sprintf(pkgKeyFormat, family, osVer, pack), k)
			}
			if _, ok := newDeps[k]; !ok {
				_ = pipe.HDel(ctx, fmt.Sprintf(defKeyFormat, family, osVer), k)
			}
		}
	}
	newDepsJSON, err := json.Marshal(newDeps)
	if err != nil {
		return xerrors.Errorf("Failed to Marshal JSON. err: %w", err)
	}
	_ = pipe.Set(ctx, depKey, string(newDepsJSON), 0)
	_ = pipe.Set(ctx, fmt.Sprintf(lastModifiedKeyFormat, family, osVer), root.Timestamp.Format("2006-01-02T15:04:05Z"), 0)
	if _, err = pipe.Exec(ctx); err != nil {
		return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
	}

	return nil
}

// CountDefs counts the number of definitions specified by args
func (r *RedisDriver) CountDefs(family, osVer string) (int, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return 0, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	count, err := r.conn.HLen(context.Background(), fmt.Sprintf(defKeyFormat, family, osVer)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return 0, xerrors.Errorf("Failed to HLen. err: %w", err)
		}
		return 0, nil
	}

	return int(count), nil
}

// GetLastModified get last modified time of OVAL in roots
func (r *RedisDriver) GetLastModified(family, osVer string) (time.Time, error) {
	family, osVer, err := formatFamilyAndOSVer(family, osVer)
	if err != nil {
		return time.Time{}, xerrors.Errorf("Failed to formatFamilyAndOSVer. err: %w", err)
	}

	lastModifiedStr, err := r.conn.Get(context.Background(), fmt.Sprintf(lastModifiedKeyFormat, family, osVer)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return time.Time{}, xerrors.Errorf("Failed to Get. err: %w", err)
		}
		return time.Now().AddDate(-100, 0, 0), nil
	}

	lastModified, err := time.Parse("2006-01-02T15:04:05Z", lastModifiedStr)
	if err != nil {
		return time.Time{}, xerrors.Errorf("Failed to parse LastModifier. err: %w", err)
	}
	return lastModified, nil
}

// IsGovalDictModelV1 determines if the DB was created at the time of goval-dictionary Model v1
func (r *RedisDriver) IsGovalDictModelV1() (bool, error) {
	ctx := context.Background()

	exists, err := r.conn.Exists(ctx, fetchMetaKey).Result()
	if err != nil {
		return false, xerrors.Errorf("Failed to Exists. err: %w", err)
	}
	if exists == 0 {
		keys, _, err := r.conn.Scan(ctx, 0, "OVAL#*", 1).Result()
		if err != nil {
			return false, xerrors.Errorf("Failed to Scan. err: %w", err)
		}
		if len(keys) == 0 {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

// GetFetchMeta get FetchMeta from Database
func (r *RedisDriver) GetFetchMeta() (*models.FetchMeta, error) {
	ctx := context.Background()

	exists, err := r.conn.Exists(ctx, fetchMetaKey).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to Exists. err: %w", err)
	}
	if exists == 0 {
		return &models.FetchMeta{GovalDictRevision: c.Revision, SchemaVersion: models.LatestSchemaVersion, LastFetchedAt: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)}, nil
	}

	revision, err := r.conn.HGet(ctx, fetchMetaKey, "Revision").Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGet Revision. err: %w", err)
	}

	verstr, err := r.conn.HGet(ctx, fetchMetaKey, "SchemaVersion").Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGet SchemaVersion. err: %w", err)
	}
	version, err := strconv.ParseUint(verstr, 10, 8)
	if err != nil {
		return nil, xerrors.Errorf("Failed to ParseUint. err: %w", err)
	}

	datestr, err := r.conn.HGet(ctx, fetchMetaKey, "LastFetchedAt").Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, xerrors.Errorf("Failed to HGet LastFetchedAt. err: %w", err)
		}
		datestr = time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	date, err := time.Parse(time.RFC3339, datestr)
	if err != nil {
		return nil, xerrors.Errorf("Failed to Parse date. err: %w", err)
	}

	return &models.FetchMeta{GovalDictRevision: revision, SchemaVersion: uint(version), LastFetchedAt: date}, nil
}

// UpsertFetchMeta upsert FetchMeta to Database
func (r *RedisDriver) UpsertFetchMeta(fetchMeta *models.FetchMeta) error {
	return r.conn.HSet(context.Background(), fetchMetaKey, map[string]interface{}{"Revision": c.Revision, "SchemaVersion": models.LatestSchemaVersion, "LastFetchedAt": fetchMeta.LastFetchedAt}).Err()
}
