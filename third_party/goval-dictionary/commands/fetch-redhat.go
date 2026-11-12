package commands

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	c "github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/db"
	fetcher "github.com/vulsio/goval-dictionary/fetcher/redhat"
	"github.com/vulsio/goval-dictionary/log"
	"github.com/vulsio/goval-dictionary/models"
	"github.com/vulsio/goval-dictionary/models/redhat"
	"github.com/vulsio/goval-dictionary/util"
)

// fetchRedHatCmd is Subcommand for fetch RedHat OVAL
var fetchRedHatCmd = &cobra.Command{
	Use:     "redhat [version]",
	Short:   "Fetch Vulnerability dictionary from RedHat",
	Long:    `Fetch Vulnerability dictionary from RedHat`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    fetchRedHat,
	Example: "$ goval-dictionary fetch redhat 8 9",
}

func init() {
	fetchCmd.AddCommand(fetchRedHatCmd)
}

func fetchRedHat(_ *cobra.Command, args []string) (err error) {
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

	results, err := fetcher.FetchFiles(util.Unique(args))
	if err != nil {
		return xerrors.Errorf("Failed to fetch files. err: %w", err)
	}

	for v, rs := range results {
		m := map[string]redhat.Root{}
		for _, r := range rs {
			ovalroot := redhat.Root{}
			if err = xml.Unmarshal(r.Body, &ovalroot); err != nil {
				return xerrors.Errorf("Failed to unmarshal xml. url: %s, err: %w", r.URL, err)
			}

			log15.Info("Fetched", "File", r.URL[strings.LastIndex(r.URL, "/")+1:], "Count", len(ovalroot.Definitions.Definitions), "Timestamp", ovalroot.Generator.Timestamp)
			ts, err := time.Parse("2006-01-02T15:04:05", ovalroot.Generator.Timestamp)
			if err != nil {
				return xerrors.Errorf("Failed to parse timestamp. url: %s, timestamp: %s, err: %w", r.URL, ovalroot.Generator.Timestamp, err)
			}
			if ts.Before(time.Now().AddDate(0, 0, -3)) {
				log15.Warn("The fetched OVAL has not been updated for 3 days, the OVAL URL may have changed, please register a GitHub issue.", "GitHub", "https://github.com/vulsio/goval-dictionary/issues", "OVAL", r.URL, "Timestamp", ovalroot.Generator.Timestamp)
			}

			m[r.URL[strings.LastIndex(r.URL, "/")+1:]] = ovalroot
		}

		roots := make([]redhat.Root, 0, len(m))
		for _, k := range []string{fmt.Sprintf("rhel-%s-including-unpatched.oval.xml.bz2", v), fmt.Sprintf("rhel-%s-extras-including-unpatched.oval.xml.bz2", v), fmt.Sprintf("rhel-%s-supplementary.oval.xml.bz2", v), fmt.Sprintf("rhel-%s-els.oval.xml.bz2", v), fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v), fmt.Sprintf("com.redhat.rhsa-RHEL%s-ELS.xml", v)} {
			roots = append(roots, m[k])
		}

		root := models.Root{
			Family:      c.RedHat,
			OSVersion:   v,
			Definitions: redhat.ConvertToModel(v, roots),
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
