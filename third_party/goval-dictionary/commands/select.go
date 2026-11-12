package commands

import (
	"errors"
	"fmt"

	"github.com/k0kubun/pp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/db"
	"github.com/vulsio/goval-dictionary/models"
)

// SelectCmd is Subcommand for fetch RedHat OVAL
var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select from DB",
	Long:  `Select from DB`,
}

func init() {
	RootCmd.AddCommand(selectCmd)

	selectCmd.AddCommand(
		&cobra.Command{
			Use:   "package <family> <release> <package name> (<arch>)",
			Short: "Select OVAL by package name",
			Args:  cobra.RangeArgs(3, 4),
			RunE: func(_ *cobra.Command, args []string) error {
				arch := ""
				if len(args) == 4 {
					switch args[0] {
					case config.Amazon, config.Oracle, config.Fedora:
						arch = args[3]
					default:
						return xerrors.Errorf("Family: %s cannot use the Architecture argument.", args[0])
					}
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
					return xerrors.Errorf("Failed to select command. err: SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
				}

				dfs, err := driver.GetByPackName(args[0], args[1], args[2], arch)
				if err != nil {
					return xerrors.Errorf("Failed to get cve by package. err: %w", err)
				}

				for _, d := range dfs {
					for _, cve := range d.Advisory.Cves {
						fmt.Printf("%s\n", cve.CveID)
						for _, pack := range d.AffectedPacks {
							fmt.Printf("    %v\n", pack)
						}
					}
				}
				fmt.Println("------------------")
				pp.ColoringEnabled = false
				_, _ = pp.Println(dfs)

				return nil
			},
			Example: `$ goval-dictionary select package ubuntu 24.04 bash
$ goval-dictionary select package oracle 9 bash x86_64`,
		},
		&cobra.Command{
			Use:   "cve-id <family> <release> <cve id> (<arch>)",
			Short: "Select OVAL by CVE-ID",
			Args:  cobra.RangeArgs(3, 4),
			RunE: func(_ *cobra.Command, args []string) error {
				arch := ""
				if len(args) == 4 {
					switch args[0] {
					case config.Amazon, config.Oracle, config.Fedora:
						arch = args[3]
					default:
						return xerrors.Errorf("Family: %s cannot use the Architecture argument.", args[0])
					}
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
					return xerrors.Errorf("Failed to select command. err: SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
				}

				dfs, err := driver.GetByCveID(args[0], args[1], args[2], arch)
				if err != nil {
					return xerrors.Errorf("Failed to get cve by cveID. err: %w", err)
				}
				for _, d := range dfs {
					fmt.Printf("%s\n", d.Title)
					fmt.Printf("%v\n", d.Advisory.Cves)
				}
				fmt.Println("------------------")
				pp.ColoringEnabled = false
				_, _ = pp.Println(dfs)

				return nil
			},
			Example: `$ goval-dictionary select cve-id ubuntu 24.04 CVE-2024-6387
$ goval-dictionary select cve-id oracle 9 CVE-2024-6387 x86_64`,
		},
		&cobra.Command{
			Use:   "advisories <family> <release>",
			Short: "List Advisories and Releated CVE-IDs",
			Args:  cobra.ExactArgs(2),
			RunE: func(_ *cobra.Command, args []string) error {
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
					return xerrors.Errorf("Failed to select command. err: SchemaVersion is old. SchemaVersion: %+v", map[string]uint{"latest": models.LatestSchemaVersion, "DB": fetchMeta.SchemaVersion})
				}

				m, err := driver.GetAdvisories(args[0], args[1])
				if err != nil {
					return xerrors.Errorf("Failed to get cve by cveID. err: %w", err)
				}
				pp.ColoringEnabled = false
				_, _ = pp.Println(m)

				return nil
			},
			Example: `$ goval-dictionary select advisories ubuntu 24.04`,
		},
	)

}
