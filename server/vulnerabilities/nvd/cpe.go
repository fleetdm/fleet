package nvd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
	"github.com/google/go-github/v37/github"
	"github.com/jmoiron/sqlx"
)

const (
	owner         = "fleetdm"
	repo          = "nvd"
	cpeDBFilename = "cpe.sqlite"
)

var cpeDBRegex = regexp.MustCompile(`^cpe-.*\.sqlite\.gz$`)

// GetGithubNVDAsset looks at the last 10 releases and returns the first (release, asset) pair that
// matches pred
func GetGithubNVDAsset(pred func(rel *github.ReleaseAsset) bool) (*github.RepositoryRelease, *github.ReleaseAsset, error) {
	ghClient := github.NewClient(fleethttp.NewGithubClient())

	releases, _, err := ghClient.Repositories.ListReleases(
		context.Background(),
		owner,
		repo,
		&github.ListOptions{Page: 0, PerPage: 10},
	)
	if err != nil {
		return nil, nil, err
	}

	for _, release := range releases {
		// skip draft releases
		if release.GetDraft() {
			continue
		}

		for _, asset := range release.Assets {
			if pred(asset) {
				return release, asset, nil
			}
		}

	}

	return nil, nil, errors.New("no nvd release found")
}

// DownloadCPEDB downloads the CPE database to the given vulnPath. If cpeDBURL is empty, attempts to download it
// from the latest release of github.com/fleetdm/nvd. Skips downloading if CPE database is newer than the release.
func DownloadCPEDBFromGithub(vulnPath string, cpeDBURL string) error {
	path := filepath.Join(vulnPath, cpeDBFilename)

	if cpeDBURL == "" {
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		case stat.ModTime().Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour)):
			// Vulnerability assets are published once per day - if the asset in question has a
			// mod date of 'today', then we can assume that is already up to day so there's nothing
			// else to do.
			return nil
		}

		rel, asset, err := GetGithubNVDAsset(func(asset *github.ReleaseAsset) bool {
			return cpeDBRegex.MatchString(asset.GetName())
		})
		if err != nil {
			return err
		}
		if asset == nil {
			return errors.New("failed to find cpe database in nvd release")
		}
		if stat != nil && stat.ModTime().After(rel.CreatedAt.Time) {
			// file is newer than release, do nothing
			return nil
		}

		cpeDBURL = asset.GetBrowserDownloadURL()
	}

	u, err := url.Parse(cpeDBURL)
	if err != nil {
		return err
	}

	githubClient := fleethttp.NewGithubClient()
	if err := download.DownloadAndExtract(githubClient, u, path); err != nil {
		return err
	}

	return nil
}

// cpeGeneralSearchQuery puts together several search statements to find the correct row in the CPE datastore.
// Each statement has a custom weight column, where 1 is the highest priority (most likely to be correct).
// The SQL statements are combined into a master statements with UNION.
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

	// 4 - Try vendor/product from bundle identifier, like tld.vendor.product
	bundleParts := strings.Split(software.BundleIdentifier, ".")
	if len(bundleParts) == 3 {
		search4 := dialect.From(goqu.I("cpe_2").As("c")).
			Select("c.rowid", "c.product", "c.vendor", "c.deprecated", goqu.L("4 as weight")).
			Where(
				goqu.L("c.vendor = ?", strings.ToLower(bundleParts[1])), goqu.L("c.product = ?", strings.ToLower(bundleParts[2])),
			)
		datasets = append(datasets, search4)
	}

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
func CPEFromSoftware(logger log.Logger, db *sqlx.DB, software *fleet.Software, translations CPETranslations, reCache *regexpCache) (string, error) {
	if containsNonASCII(software.Name) {
		level.Debug(logger).Log("msg", "skipping software with non-ascii characters", "software", software.Name, "version", software.Version, "source", software.Source)
		return "", nil
	}

	translation, match, err := translations.Translate(reCache, software)
	if err != nil {
		return "", fmt.Errorf("translate software: %w", err)
	}

	if match {
		if translation.Skip {
			level.Debug(logger).Log("msg", "CPE match skipped", "software", software.Name, "version", software.Version, "source", software.Source)
			return "", nil
		}

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
		var match *IndexedCPEItem

		err = db.Select(&results, stm, args...)
		if err == sql.ErrNoRows {
			return "", nil
		}

		if err != nil {
			return "", fmt.Errorf("getting cpes for: %s: %w", software.Name, err)
		}

		for i, item := range results {
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

			if hasAllTerms {
				match = &results[i]
				break
			}
		}

		if match != nil {
			if !match.Deprecated {
				return match.FmtStr(software), nil
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
	}

	return "", nil
}

func consumeCPEBuffer(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	batch []fleet.SoftwareCPE,
) error {
	var toDelete []fleet.SoftwareCPE
	var toUpsert []fleet.SoftwareCPE

	for i := range batch {
		// This could be because of a new translation rule or because we fixed a bug with the CPE
		// detection process
		if batch[i].CPE == "" {
			toDelete = append(toDelete, batch[i])
			continue
		}
		toUpsert = append(toUpsert, batch[i])
	}

	if len(toUpsert) != 0 {
		upserted, err := ds.UpsertSoftwareCPEs(ctx, toUpsert)
		if err != nil {
			return err
		}
		if int(upserted) != len(toUpsert) {
			level.Debug(logger).Log("toUpsert", len(toUpsert), "upserted", upserted)
		}
	}

	if len(toDelete) != 0 {
		deleted, err := ds.DeleteSoftwareCPEs(ctx, toDelete)
		if err != nil {
			return err
		}
		if int(deleted) != len(toDelete) {
			level.Debug(logger).Log("toDelete", len(toDelete), "deleted", deleted)
		}
	}

	return nil
}

func TranslateSoftwareToCPE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
) error {
	dbPath := filepath.Join(vulnPath, cpeDBFilename)

	// Skip software from sources for which we will be using OVAL for vulnerability detection.
	iterator, err := ds.AllSoftwareIterator(
		ctx,
		fleet.SoftwareIterQueryOptions{
			ExcludedSources: oval.SupportedSoftwareSources,
		},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "software iterator")
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

	var buffer []fleet.SoftwareCPE
	bufferMaxSize := 500

	for iterator.Next() {
		software, err := iterator.Value()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting value from iterator")
		}
		var cpe string
		// Skip software without version to avoid false positives in the CPE
		// matching process.
		if software.Version == "" {
			level.Debug(logger).Log(
				"msg", "skipping software without version",
				"software", software.Name,
				"source", software.Source,
			)
			// We want to continue here in case the software had an invalid CPE
			// generated by a previous version of Fleet.
		} else {
			cpe, err = CPEFromSoftware(logger, db, software, cpeTranslations, reCache)
			if err != nil {
				level.Error(logger).Log(
					"msg", "error translating to CPE, skipping",
					"software", software.Name,
					"version", software.Version,
					"source", software.Source,
					"err", err,
				)
				continue
			}
		}
		if cpe == software.GenerateCPE {
			// If the generated CPE hasn't changed from what's already stored in the DB
			// then we don't need to do anything.
			continue
		}

		buffer = append(buffer, fleet.SoftwareCPE{SoftwareID: software.ID, CPE: cpe})
		if len(buffer) == bufferMaxSize {
			if err = consumeCPEBuffer(ctx, ds, logger, buffer); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting cpe")
			}
			buffer = nil
		}
	}

	if err = consumeCPEBuffer(ctx, ds, logger, buffer); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting cpe")
	}

	if err := iterator.Err(); err != nil {
		return ctxerr.Wrap(ctx, err, "iterator contains error at the end of iteration")
	}

	return nil
}

func containsNonASCII(s string) bool {
	for _, char := range s {
		if char > unicode.MaxASCII {
			return true
		}
	}
	return false
}
