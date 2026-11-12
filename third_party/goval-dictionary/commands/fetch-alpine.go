package commands

import (
	"errors"
	"time"

	"golang.org/x/xerrors"
	yaml "gopkg.in/yaml.v2"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/db"
	fetcher "github.com/vulsio/goval-dictionary/fetcher/alpine"
	"github.com/vulsio/goval-dictionary/log"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/alpine"
	"github.com/vulsio/goval-dictionary/util"
)

// fetchAlpineCmd is Subcommand for fetch Alpine secdb
// https://secdb.alpinelinux.org/
var fetchAlpineCmd = &cobra.Command{
	Use:     "alpine [version]",
	Short:   "Fetch Vulnerability dictionary from Alpine secdb",
	Long:    `Fetch Vulnerability dictionary from Alpine secdb`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    fetchAlpine,
	Example: "$ goval-dictionary fetch alpine 3.16 3.17",
}

func init() {
	fetchCmd.AddCommand(fetchAlpineCmd)
}

func fetchAlpine(_ *cobra.Command, args []string) (err error) {
	if err := log.SetLogger(viper.GetBool("log-to-file"), viper.GetString("log-dir"), viper.GetBool("debug"), viper.GetBool("log-json")); err != nil {
		return xerrors.Errorf("Failed to SetLogger. err: %w", err)
	}

	driver, err := db.NewDB(viper.GetString("dbtype"), viper.GetString("dbpath"), viper.GetBool("debug-sql"), db.Option{})
	if err != nil {
		if errors.Is(err, db.ErrDBLocked) {
			return xerrors.Errorf("Failed to open DB. Close DB connection before fetching. err: %w", err)
		}
		return xerrors.Errorf("Failed to open DB. err: %w", err)
	}

	fetchMeta, err := driver.GetFetchMeta()
	if err != nil {
		return xerrors.Errorf("Failed to get FetchMeta from DB. err: %w", err)
	}
	if fetchMeta.OutDated() {
		return xerrors.Errorf("Failed to Insert CVEs into DB. err: SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
	}
	// If the fetch fails the first time (without SchemaVersion), the DB needs to be cleaned every time, so insert SchemaVersion.
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	results, err := fetcher.FetchFiles(util.Unique(args))
	if err != nil {
		return xerrors.Errorf("Failed to fetch files. err: %w", err)
	}

	osVerDefs := map[string][]models.Definition{}
	for _, r := range results {
		var secdb alpine.SecDB
		if err := yaml.Unmarshal(r.Body, &secdb); err != nil {
			return xerrors.Errorf("Failed to unmarshal. err: %w", err)
		}
		osVerDefs[r.Target] = append(osVerDefs[r.Target], alpine.ConvertToModel(&secdb)...)
	}

	for osVer, defs := range osVerDefs {
		root := models.Root{
			Family:      c.Alpine,
			OSVersion:   osVer,
			Definitions: defs,
			Timestamp:   time.Now(),
		}
		if err := driver.InsertOval(&root); err != nil {
			return xerrors.Errorf("Failed to insert OVAL. err: %w", err)
		}
		log15.Info("Finish", "Updated", len(root.Definitions))
	}

	fetchMeta.LastFetchedAt = time.Now()
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	return nil
}
