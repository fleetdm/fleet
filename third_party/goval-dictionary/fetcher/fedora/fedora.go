package fedora

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	"github.com/vulsio/goval-dictionary/fetcher/util"
	models "github.com/vulsio/goval-dictionary/models/fedora"
)

const (
	archX8664   = "x86_64"
	archAarch64 = "aarch64"

	pubUpdateURL     = "https://dl.fedoraproject.org/pub/fedora/linux/updates/%s/Everything/%s/repodata/repomd.xml"
	pubModuleURL     = "https://dl.fedoraproject.org/pub/fedora/linux/updates/%s/Modular/%s/repodata/repomd.xml"
	archiveUpdateURL = "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/%s/Everything/%s/repodata/repomd.xml"
	archiveModuleURL = "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/%s/Modular/%s/repodata/repomd.xml"
	bugZillaURL      = "https://bugzilla.redhat.com/show_bug.cgi?ctype=xml&id=%s"
	kojiPkgURL       = "https://kojipkgs.fedoraproject.org/packages/%s/%s/%s/files/module/modulemd.%s.txt"
)

// FetchUpdateInfosFedora fetch OVAL from Fedora
func FetchUpdateInfosFedora(versions []string) (map[string]*models.Updates, error) {
	// map[osVer][updateInfoID]models.UpdateInfo
	uinfos := make(map[string]map[string]models.UpdateInfo, len(versions))
	for _, arch := range []string{archX8664, archAarch64} {
		reqs, moduleReqs := newFedoraFetchRequests(versions, arch)
		everythingResults, err := fetchEverythingFedora(reqs)
		if err != nil {
			return nil, xerrors.Errorf("fetchEverythingFedora. err: %w", err)
		}

		moduleResults, err := fetchModulesFedora(moduleReqs, arch)
		if err != nil && !errors.Is(err, errNoUpdateInfoField) {
			return nil, xerrors.Errorf("fetchModulesFedora. err: %w", err)
		}

		for osVer, result := range mergeUpdates(everythingResults, moduleResults) {
			if _, ok := uinfos[osVer]; !ok {
				uinfos[osVer] = make(map[string]models.UpdateInfo, len(result.UpdateList))
			}
			for _, uinfo := range result.UpdateList {
				if tmp, ok := uinfos[osVer][uinfo.ID]; ok {
					uinfo.Packages = uniquePackages(append(uinfo.Packages, tmp.Packages...))
				}
				uinfos[osVer][uinfo.ID] = uinfo
			}
		}
	}

	results := map[string]*models.Updates{}
	for osver, uinfoIDs := range uinfos {
		uinfos := &models.Updates{}
		for _, uinfo := range uinfoIDs {
			uinfos.UpdateList = append(uinfos.UpdateList, uinfo)
		}
		results[osver] = uinfos
	}

	for version, v := range results {
		log15.Info(fmt.Sprintf("%d Advisories for Fedora %s Fetched", len(v.UpdateList), version))
	}

	return results, nil
}

func newFedoraFetchRequests(target []string, arch string) (reqs []util.FetchRequest, moduleReqs []util.FetchRequest) {
	for _, v := range target {
		var updateURL, moduleURL string
		n, err := strconv.Atoi(v)
		if err != nil {
			log15.Warn("Skip unknown fedora.", "version", v)
			continue
		}

		switch {
		case n < 32:
			log15.Warn("Skip fedora because no vulnerability information provided.", "version", v)
			continue
		case n < 42:
			updateURL = archiveUpdateURL
			moduleURL = archiveModuleURL
		default:
			updateURL = pubUpdateURL
			moduleURL = pubModuleURL
		}

		reqs = append(reqs, util.FetchRequest{
			Target:   v,
			URL:      fmt.Sprintf(updateURL, v, arch),
			MIMEType: util.MIMETypeXML,
		})
		moduleReqs = append(moduleReqs, util.FetchRequest{
			Target:   v,
			URL:      fmt.Sprintf(moduleURL, v, arch),
			MIMEType: util.MIMETypeXML,
		})
	}
	return
}

func fetchEverythingFedora(reqs []util.FetchRequest) (map[string]*models.Updates, error) {
	log15.Info("start fetch data from repomd.xml of non-modular package")
	feeds, err := fetchFeedFilesFedora(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch feed file, err: %w", err)
	}

	updates, err := fetchUpdateInfosFedora(feeds)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch updateinfo, err: %w", err)
	}

	results, err := parseFetchResultsFedora(updates)
	if err != nil {
		return nil, xerrors.Errorf("Failed to parse fetch results, err: %w", err)
	}

	return results, nil
}

func fetchModulesFedora(reqs []util.FetchRequest, arch string) (map[string]*models.Updates, error) {
	log15.Info("start fetch data from repomd.xml of modular")
	feeds, err := fetchModuleFeedFilesFedora(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch feed file, err: %w", err)
	}

	updates, err := fetchUpdateInfosFedora(feeds)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch updateinfo, err: %w", err)
	}

	moduleYaml, err := fetchModulesYamlFedora(feeds)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch module info, err: %w", err)
	}

	results, err := parseFetchResultsFedora(updates)
	if err != nil {
		return nil, xerrors.Errorf("Failed to parse fetch results, err: %w", err)
	}

	for version, result := range results {
		for i, update := range result.UpdateList {
			yml, ok := moduleYaml[version][update.Title]
			if !ok {
				yml, err = fetchModuleInfoFromKojiPkgs(arch, update.Title)
				if err != nil {
					return nil, xerrors.Errorf("Failed to fetch module info from kojipkgs.fedoraproject.org, err: %w", err)
				}
			}
			var pkgs []models.Package
			for _, rpm := range yml.Data.Artifacts.Rpms {
				pkg, err := rpm.NewPackageFromRpm()
				if err != nil {
					return nil, xerrors.Errorf("Failed to build package info from rpm name, err: %w", err)
				}
				pkgs = append(pkgs, pkg)
			}
			results[version].UpdateList[i].Packages = pkgs
			results[version].UpdateList[i].ModularityLabel = yml.ConvertToModularityLabel()
		}
	}
	return results, nil
}

func fetchFeedFilesFedora(reqs []util.FetchRequest) ([]util.FetchResult, error) {
	if len(reqs) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}
	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return results, nil
}

var errNoUpdateInfoField = xerrors.New("No updateinfo field in the repomd")

func fetchUpdateInfosFedora(results []util.FetchResult) ([]util.FetchResult, error) {
	log15.Info("start fetch updateinfo in repomd.xml")
	updateInfoReqs, err := extractInfoFromRepoMd(results, "updateinfo")
	if err != nil {
		return nil, xerrors.Errorf("Failed to extract updateinfo from xml, err: %w", err)
	}

	if len(updateInfoReqs) == 0 {
		return nil, errNoUpdateInfoField
	}

	results, err = util.FetchFeedFiles(updateInfoReqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return results, nil
}

// variousFlawsPattern is regexp to detect title that omit the part of CVE-IDs by finding both `...` and `various flaws`
var variousFlawsPattern = regexp.MustCompile(`.*\.\.\..*various flaws.*`)

func parseFetchResultsFedora(results []util.FetchResult) (map[string]*models.Updates, error) {
	updateInfos := make(map[string]*models.Updates, len(results))
	for _, r := range results {
		var updateInfo models.Updates
		if err := xml.NewDecoder(bytes.NewReader(r.Body)).Decode(&updateInfo); err != nil {
			return nil, xerrors.Errorf("Failed to decode XML, err: %w", err)
		}
		var securityUpdate []models.UpdateInfo
		for _, update := range updateInfo.UpdateList {
			if update.Type != "security" {
				continue
			}
			cveIDs := []string{}
			for _, ref := range update.References {
				var ids []string
				if isFedoraUpdateInfoTitleReliable(ref.Title) {
					ids = util.CveIDPattern.FindAllString(ref.Title, -1)
					if ids == nil {
						// try to correct CVE-ID from description, if title has no CVE-ID
						// NOTE: If this implementation causes the result of collecting a lot of incorrect information, fix to remove it
						ids = util.CveIDPattern.FindAllString(update.Description, -1)
					}
				} else {
					var err error
					ids, err = fetchCveIDsFromBugzilla(ref.ID)
					if err != nil {
						return nil, xerrors.Errorf("Failed to fetch CVE-IDs from bugzilla, err: %w", err)
					}
				}
				if ids != nil {
					cveIDs = append(cveIDs, ids...)
				}
			}
			update.CVEIDs = util.UniqueStrings(cveIDs)
			securityUpdate = append(securityUpdate, update)
		}
		updateInfo.UpdateList = securityUpdate
		updateInfos[r.Target] = &updateInfo
	}
	return updateInfos, nil
}

func isFedoraUpdateInfoTitleReliable(title string) bool {
	if variousFlawsPattern.MatchString(title) {
		return false
	}
	// detect unreliable CVE-ID like CVE-01-0001, CVE-aaa-bbb
	return len(util.CveIDPattern.FindAllString(title, -1)) == strings.Count(title, "CVE-")
}

func fetchModuleFeedFilesFedora(reqs []util.FetchRequest) ([]util.FetchResult, error) {
	if len(reqs) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}
	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return results, nil
}

func fetchModulesYamlFedora(results []util.FetchResult) (moduleInfosPerVersion, error) {
	log15.Info("start fetch modules.yaml in repomd.xml")
	updateInfoReqs, err := extractInfoFromRepoMd(results, "modules")
	if err != nil {
		return nil, xerrors.Errorf("Failed to extract modules from xml, err: %w", err)
	}

	if len(updateInfoReqs) == 0 {
		return nil, xerrors.New("No updateinfo field in the repomd")
	}

	results, err = util.FetchFeedFiles(updateInfoReqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch modules.yaml, err: %w", err)
	}

	yamls := make(moduleInfosPerVersion, len(results))
	for _, v := range results {
		m, err := parseModulesYamlFedora(v.Body)
		if err != nil {
			return nil, xerrors.Errorf("Failed to parse modules.yaml, err: %w", err)
		}
		yamls[v.Target] = m
	}
	return yamls, nil
}

func parseModulesYamlFedora(b []byte) (moduleInfosPerPackage, error) {
	modules := moduleInfosPerPackage{}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	var contents []string
	for scanner.Scan() {
		str := scanner.Text()
		switch str {
		case "---":
			{
				contents = []string{}
			}
		case "...":
			{
				var module moduleInfo
				if err := yaml.NewDecoder(strings.NewReader(strings.Join(contents, "\n"))).Decode(&module); err != nil {
					return nil, xerrors.Errorf("failed to decode module info. err: %w", err)
				}
				if module.Version == 2 {
					modules[module.ConvertToUpdateInfoTitle()] = module
				}
			}
		default:
			{
				contents = append(contents, str)
			}
		}
	}

	return modules, nil
}

func fetchCveIDsFromBugzilla(id string) ([]string, error) {
	req := util.FetchRequest{
		URL:           fmt.Sprintf(bugZillaURL, id),
		LogSuppressed: true,
		MIMEType:      util.MIMETypeXML,
	}
	log15.Info("Fetch CVE-ID list from bugzilla.redhat.com", "URL", req.URL)
	body, err := util.FetchFeedFiles([]util.FetchRequest{req})
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch CVE-ID list, err: %w", err)
	}

	var b bugzillaXML
	if err = xml.Unmarshal(body[0].Body, &b); err != nil {
		return nil, xerrors.Errorf("Failed to unmarshal xml. url: %s, err: %w", req.URL, err)
	}

	reqs := make([]util.FetchRequest, len(b.Blocked))
	for i, v := range b.Blocked {
		reqs[i] = util.FetchRequest{
			URL:           fmt.Sprintf(bugZillaURL, v),
			LogSuppressed: true,
			MIMEType:      util.MIMETypeXML,
		}
	}

	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch CVE-IDs, err: %w", err)
	}

	var ids []string
	for _, result := range results {
		var b bugzillaXML
		if err = xml.Unmarshal(result.Body, &b); err != nil {
			return nil, xerrors.Errorf("Failed to unmarshal xml. url: %s, err: %w", req.URL, err)
		}

		if b.Alias != "" {
			ids = append(ids, b.Alias)
		}
	}

	log15.Info(fmt.Sprintf("%d CVE-IDs fetched", len(ids)))
	return ids, nil
}

func extractInfoFromRepoMd(results []util.FetchResult, rt string) ([]util.FetchRequest, error) {
	var updateInfoReqs []util.FetchRequest
	for _, r := range results {
		var repoMd repoMd
		if err := xml.NewDecoder(bytes.NewBuffer(r.Body)).Decode(&repoMd); err != nil {
			return nil, xerrors.Errorf("Failed to decode repomd of version %s. err: %w", r.Target, err)
		}

		for _, repo := range repoMd.RepoList {
			if repo.Type != rt {
				continue
			}
			u, err := url.Parse(r.URL)
			if err != nil {
				return nil, xerrors.Errorf("Failed to parse URL in XML. err: %w", err)
			}
			u.Path = strings.Replace(u.Path, "repodata/repomd.xml", repo.Location.Href, 1)

			mt, err := func() (util.MIMEType, error) {
				switch ext := filepath.Ext(path.Base(repo.Location.Href)); ext {
				case ".gz":
					return util.MIMETypeGzip, nil
				case ".xz":
					return util.MIMETypeXz, nil
				case ".zst":
					return util.MIMETypeZst, nil
				default:
					return util.MIMETypeUnknown, xerrors.Errorf("%q is not supported extension", ext)
				}
			}()
			if err != nil {
				return nil, xerrors.Errorf("Failed to get MIME Type. err: %w", err)
			}

			req := util.FetchRequest{
				URL:      u.String(),
				Target:   r.Target,
				MIMEType: mt,
			}
			updateInfoReqs = append(updateInfoReqs, req)
			break
		}
	}
	return updateInfoReqs, nil
}

// uinfoTitle is expected title of xml format as ${name}-${stream}-${version}.${context}
func fetchModuleInfoFromKojiPkgs(arch, uinfoTitle string) (moduleInfo, error) {
	req, err := newKojiPkgsRequest(arch, uinfoTitle)
	if err != nil {
		return moduleInfo{}, xerrors.Errorf("Failed to generate request to kojipkgs.fedoraproject.org, err: %w", err)
	}
	result, err := util.FetchFeedFiles([]util.FetchRequest{req})
	if err != nil {
		return moduleInfo{}, xerrors.Errorf("Failed to fetch from kojipkgs.fedoraproject.org, err: %w", err)
	}
	moduleYaml, err := parseModulesYamlFedora(result[0].Body)
	if err != nil {
		return moduleInfo{}, xerrors.Errorf("Failed to parse module text, err: %w", err)
	}
	if yml, ok := moduleYaml[uinfoTitle]; !ok {
		return yml, nil
	}
	return moduleInfo{}, xerrors.New("Module not found in kojipkgs.fedoraproject.org")
}

func newKojiPkgsRequest(arch, uinfoTitle string) (util.FetchRequest, error) {
	relIndex := strings.LastIndex(uinfoTitle, "-")
	if relIndex == -1 {
		return util.FetchRequest{}, xerrors.Errorf("Failed to parse release from title of updateinfo: %s", uinfoTitle)
	}
	rel := uinfoTitle[relIndex+1:]

	verIndex := strings.LastIndex(uinfoTitle[:relIndex], "-")
	if verIndex == -1 {
		return util.FetchRequest{}, xerrors.Errorf("Failed to parse version from title of updateinfo: %s", uinfoTitle)
	}
	ver := uinfoTitle[verIndex+1 : relIndex]
	name := uinfoTitle[:verIndex]

	req := util.FetchRequest{
		URL:      fmt.Sprintf(kojiPkgURL, name, ver, rel, arch),
		MIMEType: util.MIMETypeTxt,
	}
	return req, nil
}
