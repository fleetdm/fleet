package nvd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v37/github"
	"github.com/jmoiron/sqlx"
)

const (
	owner         = "fleetdm"
	repo          = "nvd"
	cpeDBFilename = "cpe.sqlite"
)

var cpeDBRegex = regexp.MustCompile(`^cpe-.*\.sqlite\.gz$`)

func GetLatestNVDRelease(client *http.Client) (*github.RepositoryRelease, error) {
	ghclient := github.NewClient(client)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 10})
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		// skip draft releases
		if !release.GetDraft() {
			return release, nil
		}
	}

	return nil, errors.New("no nvd release found")
}

// DownloadCPEDB downloads the CPE database to the given vulnPath. If cpeDBURL is empty, attempts to download it
// from the latest release of github.com/fleetdm/nvd. Skips downloading if CPE database is newer than the release.
func DownloadCPEDB(
	vulnPath string,
	client *http.Client,
	cpeDBURL string,
) error {
	path := filepath.Join(vulnPath, cpeDBFilename)

	if cpeDBURL == "" {
		release, err := GetLatestNVDRelease(client)
		if err != nil {
			return err
		}
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		default:
			if stat.ModTime().After(release.CreatedAt.Time) {
				// file is newer than release, do nothing
				return nil
			}
		}

		for _, asset := range release.Assets {
			if cpeDBRegex.MatchString(asset.GetName()) {
				cpeDBURL = asset.GetBrowserDownloadURL()
				break
			}
		}
		if cpeDBURL == "" {
			return errors.New("failed to find cpe database in nvd release")
		}
	}

	u, err := url.Parse(cpeDBURL)
	if err != nil {
		return err
	}
	if err := download.DownloadAndExtract(client, u, path); err != nil {
		return err
	}

	return nil
}

func cpeGeneralSearchQuery(software *fleet.Software) (string, []interface{}, error) {
	dialect := goqu.Dialect("sqlite")

	// 1 - Try to match product and vendor terms
	search1 := dialect.From(goqu.I("cpe_2").As("c")).
		Select("c.rowid", "c.product", "c.vendor", "c.deprecated", goqu.L("1 as weight"))
	var vexps []goqu.Expression
	for _, v := range vendorVariations(software) {
		vexps = append(vexps, goqu.I("c.vendor").Eq(v))
	}
	search1 = search1.Where(goqu.Or(vexps...))
	var nexps []goqu.Expression
	for _, v := range productVariations(software) {
		nexps = append(nexps, goqu.I("c.product").Eq(v))
	}
	search1 = search1.Where(goqu.Or(nexps...))

	// 2 - Try to match product only
	search2 := dialect.From(goqu.I("cpe_2").As("c")).
		Select("c.rowid", "c.product", "c.vendor", "c.deprecated", goqu.L("2 as weight")).
		Where(goqu.L("c.product = ?", sanitizeSoftwareName(software)))

	// 3 - Try Full text match
	search3 := dialect.From(goqu.I("cpe_2").As("c")).
		Select("c.rowid", "c.product", "c.vendor", "c.deprecated", goqu.L("3 as weight")).
		Join(
			goqu.I("cpe_search").As("cs"),
			goqu.On(goqu.I("cs.rowid").Eq(goqu.I("c.rowid"))),
		).
		Where(goqu.L("cs.title MATCH ?", sanitizeMatch(software.Name)))

	datasets := []*goqu.SelectDataset{search1, search2, search3}

	var sqlParts []string
	var args []interface{}
	var stm string

	for _, d := range datasets {
		s, a, err := d.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("sql: %w", err)
		}
		sqlParts = append(sqlParts, s)
		args = append(args, a...)
	}

	stm = strings.Join(sqlParts, " UNION ")
	stm += "ORDER BY weight ASC"

	return stm, args, nil
}

// CPEFromSoftware attempts to find a matching cpe entry for the given software in the NVD CPE dictionary. `db` contains data from the NVD CPE dictionary
// and is optimized for lookups, see `GenerateCPEDB`. `translations` are used to aid in cpe matching. When searching for cpes, we first check if it matches
// any translations, and then lookup in the cpe database based on the title, product and vendor.
func CPEFromSoftware(db *sqlx.DB, software *fleet.Software, translations CPETranslations, reCache *regexpCache) (string, error) {
	translation, match, err := translations.Translate(reCache, software)
	if err != nil {
		return "", fmt.Errorf("translate software: %w", err)
	}
	if match {
		ds := goqu.Dialect("sqlite").From(goqu.I("cpe_2").As("c")).
			Select(
				"c.rowid",
				"c.product",
				"c.vendor",
				"c.deprecated",
				goqu.L("1 as weight"),
			).Limit(1)

		if len(translation.Product) > 0 {
			var exps []goqu.Expression
			for _, product := range translation.Product {
				exps = append(exps, goqu.I("c.product").Eq(product))
			}
			ds = ds.Where(goqu.Or(exps...))
		}
		if len(translation.Vendor) > 0 {
			var exps []goqu.Expression
			for _, vendor := range translation.Vendor {
				exps = append(exps, goqu.I("c.vendor").Eq(vendor))
			}
			ds = ds.Where(goqu.Or(exps...))
		}
		if len(translation.TargetSW) > 0 {
			var exps []goqu.Expression
			for _, targetSW := range translation.TargetSW {
				exps = append(exps, goqu.I("c.target_sw").Eq(targetSW))
			}
			ds = ds.Where(goqu.Or(exps...))
		}

		stm, args, _ := ds.ToSQL()

		var result IndexedCPEItem
		err = db.Get(&result, stm, args...)
		if err != nil {
			return "", fmt.Errorf("getting CPE for: %s: %w", software.Name, err)
		}

		if result.ID != 0 {
			return result.FmtStr(software), nil
		}
	} else {
		stm, args, err := cpeGeneralSearchQuery(software)
		if err != nil {
			return "", fmt.Errorf("getting cpes for: %s: %w", software.Name, err)
		}

		var results []IndexedCPEItem
		err = db.Select(&results, stm, args...)
		if err == sql.ErrNoRows {
			return "", nil
		}

		if err != nil {
			return "", fmt.Errorf("getting cpes for: %s: %w", software.Name, err)
		}

		for _, item := range results {
			if !item.Deprecated {
				hasAllTerms := true

				sName := strings.ToLower(software.Name)
				for _, sN := range strings.Split(item.Product, "_") {
					hasAllTerms = hasAllTerms && strings.Index(sName, sN) != -1
				}

				sVendor := strings.ToLower(software.Vendor)
				sBundle := strings.ToLower(software.BundleIdentifier)
				for _, sV := range strings.Split(item.Vendor, "_") {
					if sVendor != "" {
						hasAllTerms = hasAllTerms && strings.Index(sVendor, sV) != -1
					}

					if sBundle != "" {
						hasAllTerms = hasAllTerms && strings.Index(sBundle, sV) != -1
					}
				}

				if !hasAllTerms {
					continue
				}

				return item.FmtStr(software), nil
			}
		}

		// try to find a non-deprecated cpe by looking up deprecated_by
		for _, item := range results {
			deprecatedItem := item
			for {
				var deprecation IndexedCPEItem

				err = db.Get(
					&deprecation,
					`
						SELECT
							rowid,
							product,
							vendor,
							deprecated
						FROM
							cpe_2
						WHERE
							cpe23 IN (
								SELECT cpe23 FROM deprecated_by d WHERE d.cpe_id = ?
							)
					`,
					deprecatedItem.ID,
				)
				if err == sql.ErrNoRows {
					break
				}
				if err != nil {
					return "", fmt.Errorf("getting deprecation: %w", err)
				}
				if deprecation.Deprecated {
					deprecatedItem = deprecation
					continue
				}

				return deprecation.FmtStr(software), nil
			}
		}
	}

	return "", nil
}

func TranslateSoftwareToCPE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
) error {
	dbPath := filepath.Join(vulnPath, cpeDBFilename)

	// Skip software from sources for which we will be using OVAL for vulnerability detection.
	iterator, err := ds.AllSoftwareWithoutCPEIterator(ctx, oval.SupportedSoftwareSources)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "all software iterator")
	}
	defer iterator.Close()

	db, err := sqliteDB(dbPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "opening the cpe db")
	}
	defer db.Close()

	cpeTranslationsPath := filepath.Join(vulnPath, cpeTranslationsFilename)
	cpeTranslations, err := loadCPETranslations(cpeTranslationsPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to load cpe translations", "err", err)
	}

	reCache := newRegexpCache()

	for iterator.Next() {
		software, err := iterator.Value()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting value from iterator")
		}
		cpe, err := CPEFromSoftware(db, software, cpeTranslations, reCache)
		if err != nil {
			level.Error(logger).Log("software->cpe", "error translating to CPE, skipping...", "err", err)
			continue
		}
		if cpe == "" {
			continue
		}
		err = ds.AddCPEForSoftware(ctx, *software, cpe)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting cpe")
		}
	}

	return nil
}
