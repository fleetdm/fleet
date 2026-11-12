package commands

import (
	"encoding/xml"
	"errors"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/db"
	fetcher "github.com/vulsio/goval-dictionary/fetcher/suse"
	"github.com/vulsio/goval-dictionary/log"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/suse"
	"github.com/vulsio/goval-dictionary/util"
)

// fetchSUSECmd is Subcommand for fetch SUSE OVAL
var fetchSUSECmd = &cobra.Command{
	Use:   "suse [version]",
	Short: "Fetch Vulnerability dictionary from SUSE",
	Long:  `Fetch Vulnerability dictionary from SUSE`,
	RunE:  fetchSUSE,
	Example: `$ goval-dictionary fetch suse --suse-type opensuse 13.2 tumbleweed
$ goval-dictionary fetch suse --suse-type opensuse-leap 15.2 15.3
$ goval-dictionary fetch suse --suse-type suse-enterprise-server 12 15
$ goval-dictionary fetch suse --suse-type suse-enterprise-desktop 12 15`,
}

func init() {
	fetchCmd.AddCommand(fetchSUSECmd)

	fetchSUSECmd.PersistentFlags().String("suse-type", "opensuse-leap", "Fetch SUSE Type(choices: opensuse, opensuse-leap, suse-enterprise-server, suse-enterprise-desktop)")
	_ = viper.BindPFlag("suse-type", fetchSUSECmd.PersistentFlags().Lookup("suse-type"))
}

func fetchSUSE(_ *cobra.Command, args []string) (err error) {
	if err := log.SetLogger(viper.GetBool("log-to-file"), viper.GetString("log-dir"), viper.GetBool("debug"), viper.GetBool("log-json")); err != nil {
		return xerrors.Errorf("Failed to SetLogger. err: %w", err)
	}

	var suseType string
	switch viper.GetString("suse-type") {
	case "opensuse":
		suseType = c.OpenSUSE
	case "opensuse-leap":
		suseType = c.OpenSUSELeap
	case "suse-enterprise-server":
		suseType = c.SUSEEnterpriseServer
	case "suse-enterprise-desktop":
		suseType = c.SUSEEnterpriseDesktop
	default:
		return xerrors.Errorf("Specify SUSE type to fetch. Available SUSE Type: opensuse, opensuse-leap, suse-enterprise-server, suse-enterprise-desktop")
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

	results, err := fetcher.FetchFiles(suseType, util.Unique(args))
	if err != nil {
		return xerrors.Errorf("Failed to fetch files. err: %w", err)
	}

	for _, r := range results {
		ovalroot := suse.Root{}
		if err = xml.Unmarshal(r.Body, &ovalroot); err != nil {
			return xerrors.Errorf("Failed to unmarshal xml. url: %s, err: %w", r.URL, err)
		}
		filename := r.URL[strings.LastIndex(r.URL, "/")+1:]
		log15.Info("Fetched", "File", filename, "Count", len(ovalroot.Definitions.Definitions), "Timestamp", ovalroot.Generator.Timestamp)
		ts, err := time.Parse("2006-01-02T15:04:05", ovalroot.Generator.Timestamp)
		if err != nil {
			return xerrors.Errorf("Failed to parse timestamp. url: %s, timestamp: %s, err: %w", r.URL, ovalroot.Generator.Timestamp, err)
		}
		if ts.Before(time.Now().AddDate(0, 0, -3)) {
			log15.Warn("The fetched OVAL has not been updated for 3 days, the OVAL URL may have changed, please register a GitHub issue.", "GitHub", "https://github.com/vulsio/goval-dictionary/issues", "OVAL", r.URL, "Timestamp", ovalroot.Generator.Timestamp)
		}

		osVerDefs, err := suse.ConvertToModel(filename, &ovalroot)
		if err != nil {
			return xerrors.Errorf("Failed to convert from OVAL to goval-dictionary model. err: %w", err)
		}
		for osVer, defs := range osVerDefs {
			root := models.Root{
				Family:      suseType,
				OSVersion:   osVer,
				Definitions: defs,
				Timestamp:   time.Now(),
			}
			if err := driver.InsertOval(&root); err != nil {
				return xerrors.Errorf("Failed to insert OVAL. err: %w", err)
			}
			log15.Info("Finish", "Updated", len(root.Definitions))
		}
	}

	fetchMeta.LastFetchedAt = time.Now()
	if err := driver.UpsertFetchMeta(fetchMeta); err != nil {
		return xerrors.Errorf("Failed to upsert FetchMeta to DB. err: %w", err)
	}

	return nil
}
