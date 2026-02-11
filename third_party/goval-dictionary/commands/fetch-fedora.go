package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/db"
	fetcher "github.com/vulsio/goval-dictionary/fetcher/fedora"
	"github.com/vulsio/goval-dictionary/log"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/fedora"
	"github.com/vulsio/goval-dictionary/util"
)

// fetchFedoraCmd is Subcommand for fetch Fedora OVAL
var fetchFedoraCmd = &cobra.Command{
	Use:     "fedora [version]",
	Short:   "Fetch Vulnerability dictionary from Fedora",
	Long:    `Fetch Vulnerability dictionary from Fedora`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    fetchFedora,
	Example: "$ goval-dictionary fetch fedora 37",
}

func init() {
	fetchCmd.AddCommand(fetchFedoraCmd)
}

func fetchFedora(_ *cobra.Command, args []string) (err error) {
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
		return xerrors.Errorf("Failed to Insert CVEs into DB. SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
	}
	// If the fetch fails the first time (without SchemaVersion), the DB needs to be cleaned every time, so insert SchemaVersion.
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	uinfos, err := fetcher.FetchUpdateInfosFedora(util.Unique(args))
	if err != nil {
		return xerrors.Errorf("Failed to fetch files. err: %w", err)
	}

	for k, v := range uinfos {
		root := models.Root{
			Family:      c.Fedora,
			OSVersion:   k,
			Definitions: fedora.ConvertToModel(v),
			Timestamp:   time.Now(),
		}
		log15.Info(fmt.Sprintf("%d CVEs for Fedora %s. Inserting to DB", len(root.Definitions), k))
		if err := driver.InsertOval(&root); err != nil {
			return xerrors.Errorf("Failed to insert OVAL. err: %w", err)
		}
		log15.Info("Finish", "Updated", len(root.Definitions))
	}
	return nil
}
