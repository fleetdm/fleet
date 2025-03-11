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
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
		return fmt.Errorf("download and extract %s: %w", u.String(), err)
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

// softwareTransformers provide logic for tweaking e.g. software versions to match what's in the NVD database. These
// changes are done here rather than in sanitizeSoftware to ensure that software versions visible in the UI are the
// raw version strings.
var (
	macOSMSTeamsVersion  = regexp.MustCompile(`(\d).00.(\d)(\d+)`)
	citrixName           = regexp.MustCompile(`Citrix Workspace [0-9]+`)
	minioAltDate         = regexp.MustCompile(`^\d{14}$`)
	softwareTransformers = []struct {
		matches func(*fleet.Software) bool
		mutate  func(*fleet.Software, log.Logger)
	}{
		{
			// JetBrains EAP version numbers aren't what are used in CPEs; this handles the translation for Mac versions.
			// See #22723 for background. Bundle identifier for EAPs also ends with "-EAP" but checking version makes it
			// a bit easier to add other platforms later. EAP version numbers are e.g. EAP GO-243.21565.42, and checking
			// here for the dash ensures that string splitting in the mutator always works without a bounds check.
			matches: func(s *fleet.Software) bool {
				return s.BundleIdentifier != "" && strings.HasPrefix(s.BundleIdentifier, "com.jetbrains.") &&
					strings.HasPrefix(s.Version, "EAP ") && strings.Contains(s.Version, "-")
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				// 243 -> 2024.3
				eapMajorVersion := strings.Split(strings.Split(s.Version, "-")[1], ".")[0]
				yearBasedMajorVersion, err := strconv.Atoi("20" + eapMajorVersion[:2])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse JetBrains EAP major version", "version", s.Version, "err", err)
					return
				}
				yearBasedMinorVersion, err := strconv.Atoi(eapMajorVersion[2:])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse JetBrains EAP minor version", "version", s.Version, "err", err)
					return
				}

				// EAPs are treated as having all fixes from the previous year-based release, but no fixes from the
				// year-based release they're an EAP of. The exception to this would be CVE-2024-37051, which was fixed
				// in a second/third EAP depending on product, but at this point all vulnerable EAPs force exit on
				// startup due to being expired, so that CVE can't be exploited.
				yearBasedMinorVersion -= 1
				if yearBasedMinorVersion <= 0 { // wrap e.g. 2024.1 to 2023.4 (not a real version, but has all 2023.3 fixes)
					yearBasedMajorVersion -= 1
					yearBasedMinorVersion = 4
				}

				// pass through minor and patch version for EAP to tell different EAP builds apart
				eapMinorAndPatchVersion := strings.Join(strings.Split(strings.Split(s.Version, "-")[1], ".")[1:], ".")
				s.Version = fmt.Sprintf("%d.%d.%s.%s", yearBasedMajorVersion, yearBasedMinorVersion, "99", eapMinorAndPatchVersion)
			},
		},
		{
			matches: func(s *fleet.Software) bool {
				return s.Source == "programs" && strings.HasPrefix(s.Name, "Python 3.")
			},
			mutate: func(s *fleet.Software, logger kitlog.Logger) {
				versionComponents := strings.Split(s.Version, ".")
				// Python 3 versions on Windows should always look like 3.14.102.0; if they don't we
				// should bail out to avoid bad indexing panics.
				if len(versionComponents) < 4 {
					level.Debug(logger).Log("msg", "expected 4 version components", "gotCount", len(versionComponents))
					return
				}
				if len(versionComponents[2]) < 3 {
					level.Debug(logger).Log("msg", "got a patch version component with unexpected length", "gotPatchVersion", versionComponents[2])
					return
				}
				patchVersion := versionComponents[2][0 : len(versionComponents[2])-3]
				releaseLevel := versionComponents[2][len(versionComponents[2])-3 : len(versionComponents[2])-1]
				releaseSerial := versionComponents[2][len(versionComponents[2])-1 : len(versionComponents[2])]

				candidateSuffix := ""
				switch releaseLevel { // see https://github.com/python/cpython/issues/100829#issuecomment-1374656643
				case "10":
					candidateSuffix = "a" + releaseSerial
				case "11":
					candidateSuffix = "b" + releaseSerial
				case "12":
					candidateSuffix = "rc" + releaseSerial
				} // default

				if patchVersion == "" { // dot-zero patch releases have a 3-digit patch + build number
					patchVersion = "0"
				}

				versionComponents[2] = patchVersion + candidateSuffix
				s.Version = strings.Join(versionComponents[0:3], ".")
			},
		},
		{
			matches: func(s *fleet.Software) bool {
				return s.Name == "Cloudflare WARP" && s.Source == "programs"
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				// Perform some sanity check on the version before mutating it.
				parts := strings.Split(s.Version, ".")
				if len(parts) <= 1 {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version)
					return
				}
				_, err := strconv.Atoi(parts[0])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}
				// In case Cloudflare starts returning the full year.
				if len(parts[0]) == 4 {
					return
				}
				s.Version = "20" + s.Version // Cloudflare WARP was released on 2019.
			},
		},
		{
			matches: func(s *fleet.Software) bool {
				return s.Source == "apps" && (s.Name == "Microsoft Teams.app" || s.Name == "Microsoft Teams classic.app")
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				if matches := macOSMSTeamsVersion.FindStringSubmatch(s.Version); len(matches) > 0 {
					s.Version = fmt.Sprintf("%s.%s.00.%s", matches[1], matches[2], matches[3])
				}
			},
		},
		{
			matches: func(s *fleet.Software) bool {
				return citrixName.Match([]byte(s.Name)) || s.Name == "Citrix Workspace.app"
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				parts := strings.Split(s.Version, ".")
				if len(parts) <= 1 {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version)
					return
				}

				if len(parts[0]) > 2 {
					// then the versioning is correct, so no need to change
					return
				}

				part1, err := strconv.Atoi(parts[0])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}

				part2, err := strconv.Atoi(parts[1])
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}

				newFirstPart := part1*100 + part2
				newFirstStr := strconv.Itoa(newFirstPart)
				newParts := []string{newFirstStr}
				newParts = append(newParts, parts[2:]...)
				s.Version = strings.Join(newParts, ".")
			},
		},
		{
			// Trim the "RELEASE." prefix from Minio versions.
			matches: func(s *fleet.Software) bool {
				return s.Name == "minio" && strings.Contains(s.Version, "RELEASE.")
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				// trim the "RELEASE." prefix from the version
				s.Version = strings.TrimPrefix(s.Version, "RELEASE.")
				// trim any unexpected trailing characters
				if idx := strings.Index(s.Version, "_"); idx != -1 {
					s.Version = s.Version[:idx]
				}
			},
		},
		{
			// Convert the timestamp to NVD's format for Minio versions.
			matches: func(s *fleet.Software) bool {
				return s.Name == "minio" && minioAltDate.MatchString(s.Version)
			},
			mutate: func(s *fleet.Software, logger log.Logger) {
				timestamp, err := time.Parse("20060102150405", s.Version)
				if err != nil {
					level.Debug(logger).Log("msg", "failed to parse software version", "name", s.Name, "version", s.Version, "err", err)
					return
				}
				s.Version = timestamp.Format("2006-01-02T15-04-05Z")
			},
		},
	}
)

func mutateSoftware(software *fleet.Software, logger log.Logger) {
	for _, transformer := range softwareTransformers {
		if transformer.matches(software) {
			defer func() {
				if r := recover(); r != nil {
					level.Warn(logger).Log("msg", "panic during software mutation", "softwareName", software.Name, "softwareVersion", software.Version, "error", r)
				}
			}()
			transformer.mutate(software, logger)
			break
		}
	}
}

// CPEFromSoftware attempts to find a matching cpe entry for the given software in the NVD CPE dictionary. `db` contains data from the NVD CPE dictionary
// and is optimized for lookups, see `GenerateCPEDB`. `translations` are used to aid in cpe matching. When searching for cpes, we first check if it matches
// any translations, and then lookup in the cpe database based on the title, product and vendor.
func CPEFromSoftware(logger log.Logger, db *sqlx.DB, software *fleet.Software, translations CPETranslations, reCache *regexpCache) (string, error) {
	if containsNonASCII(software.Name) {
		level.Debug(logger).Log("msg", "skipping software with non-ascii characters", "software", software.Name, "version", software.Version, "source", software.Source)
		return "", nil
	}

	mutateSoftware(software, logger) // tweak e.g. software versions prior to CPE matching if needed

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
			if translation.Part != "" {
				result.Part = translation.Part
			}
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
				hasAllTerms = hasAllTerms && strings.Contains(sName, sN)
			}

			sVendor := strings.ToLower(software.Vendor)
			sBundle := strings.ToLower(software.BundleIdentifier)
			for _, sV := range strings.Split(item.Vendor, "_") {
				if sVendor != "" {
					hasAllTerms = hasAllTerms && strings.Contains(sVendor, sV)
				}

				if sBundle != "" {
					hasAllTerms = hasAllTerms && strings.Contains(sBundle, sV)
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

// mysql 5.7 compatible regexp for ubuntu kernel package names
const LinuxImageRegex = `^linux-image-[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+-[[:digit:]]+-[[:alnum:]]+`

// knownUbuntuKernelVariants is a list of known kernel variants that are used in the Ubuntu kernel
// OVAL feeds.  These are used to determine if a kernel package is a custom variant and should be
// matched against the NVD feed rather than the OVAL feed.
var knownUbuntuKernelVariants = []string{
	"allwinner",
	"aws",
	"aws-hwe",
	"azure",
	"azure-fde",
	"bluefield",
	"dell300x",
	"euclid",
	"gcp",
	"generic",
	"generic-64k",
	"generic-lpae",
	"gke",
	"gkeop",
	"intel",
	"intel-iotg",
	"ibm",
	"iot",
	"kvm",
	"laptop",
	"lowlatency",
	"lowlatency-64k",
	"nvidia",
	"nvidia-64k",
	"nvidia-lowlatency",
	"oem",
	"oem-osp1",
	"oracle",
	"oracle-64k",
	"powerpc-e500",
	"powerpc-e500mc",
	"powerpc-smp",
	"powerpc64-emb",
	"powerpc64-smp",
	"raspi",
	"raspi-nolpae",
	"raspi2",
	"snapdragon",
	"starfive",
	"xilinx-zynqmp",
}

func BuildLinuxExclusionRegex() string {
	return fmt.Sprintf("-(%s)$", strings.Join(knownUbuntuKernelVariants, "|"))
}

func TranslateSoftwareToCPE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
) error {
	// Skip software from sources for which we will be using OVAL or goval-dictionary for vulnerability detection.
	nonOvalIterator, err := ds.AllSoftwareIterator(
		ctx,
		fleet.SoftwareIterQueryOptions{
			// Also exclude iOS and iPadOS apps until we enable vulnerabilities support for them.
			ExcludedSources: append(oval.SupportedSoftwareSources, "ios_apps", "ipados_apps"),
		},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "non-oval software iterator")
	}
	defer nonOvalIterator.Close()

	err = translateSoftwareToCPEWithIterator(ctx, ds, vulnPath, logger, nonOvalIterator)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "translate non-oval software to CPE")
	}

	if err := nonOvalIterator.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing non-oval software iterator")
	}

	ubuntuKernelIterator, err := ds.AllSoftwareIterator(
		ctx,
		fleet.SoftwareIterQueryOptions{
			IncludedSources: []string{"deb_packages"},
			NameMatch:       LinuxImageRegex,
			NameExclude:     BuildLinuxExclusionRegex(),
		},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ubuntu kernel iterator")
	}
	defer ubuntuKernelIterator.Close()

	err = translateSoftwareToCPEWithIterator(ctx, ds, vulnPath, logger, ubuntuKernelIterator)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "translate ubuntu kernel to CPE")
	}

	if err := ubuntuKernelIterator.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing ubuntu kernel iterator")
	}

	return nil
}

func translateSoftwareToCPEWithIterator(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	iterator fleet.SoftwareIterator,
) error {
	dbPath := filepath.Join(vulnPath, cpeDBFilename)

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

var allowedNonASCII = []int32{
	'–', // en dash
	'—', // em dash
}

func containsNonASCII(s string) bool {
	for _, char := range s {
		if char > unicode.MaxASCII && !slices.Contains(allowedNonASCII, char) {
			return true
		}
	}
	return false
}
