package fedora

import (
	"fmt"
	"strings"

	"golang.org/x/xerrors"

	models "github.com/vulsio/goval-dictionary/models/fedora"
)

// repoMd has repomd data
type repoMd struct {
	RepoList []repo `xml:"data"`
}

// repo has a repo data
type repo struct {
	Type     string   `xml:"type,attr"`
	Location location `xml:"location"`
}

// location has a location of repomd
type location struct {
	Href string `xml:"href,attr"`
}

type bugzillaXML struct {
	Blocked []string `xml:"bug>blocked" json:"blocked,omitempty"`
	Alias   string   `xml:"bug>alias" json:"alias,omitempty"`
}

// moduleInfo has a data of modules.yaml
type moduleInfo struct {
	Version int `yaml:"version"`
	Data    struct {
		Name      string `yaml:"name"`
		Stream    string `yaml:"stream"`
		Version   int64  `yaml:"version"`
		Context   string `yaml:"context"`
		Arch      string `yaml:"arch"`
		Artifacts struct {
			Rpms []Rpm `yaml:"rpms"`
		} `yaml:"artifacts"`
	} `yaml:"data"`
}

type moduleInfosPerVersion map[string]moduleInfosPerPackage

type moduleInfosPerPackage map[string]moduleInfo

// ConvertToUpdateInfoTitle generates file name from data of modules.yaml
func (f moduleInfo) ConvertToUpdateInfoTitle() string {
	return fmt.Sprintf("%s-%s-%d.%s", f.Data.Name, f.Data.Stream, f.Data.Version, f.Data.Context)
}

// ConvertToModularityLabel generates modularity_label from data of modules.yaml
func (f moduleInfo) ConvertToModularityLabel() string {
	return fmt.Sprintf("%s:%s:%d:%s", f.Data.Name, f.Data.Stream, f.Data.Version, f.Data.Context)
}

// Rpm is a package name of data/artifacts/rpms in modules.yaml
type Rpm string

// NewPackageFromRpm generates Package{} by parsing package name
func (r Rpm) NewPackageFromRpm() (models.Package, error) {
	filename := strings.TrimSuffix(string(r), ".rpm")

	archIndex := strings.LastIndex(filename, ".")
	if archIndex == -1 {
		return models.Package{}, xerrors.Errorf("Failed to parse arch from filename: %s", filename)
	}
	arch := filename[archIndex+1:]

	relIndex := strings.LastIndex(filename[:archIndex], "-")
	if relIndex == -1 {
		return models.Package{}, xerrors.Errorf("Failed to parse release from filename: %s", filename)
	}
	rel := filename[relIndex+1 : archIndex]

	verIndex := strings.LastIndex(filename[:relIndex], "-")
	if verIndex == -1 {
		return models.Package{}, xerrors.Errorf("Failed to parse version from filename: %s", filename)
	}
	ver := filename[verIndex+1 : relIndex]

	epochIndex := strings.Index(ver, ":")
	var epoch string
	if epochIndex == -1 {
		epoch = "0"
	} else {
		epoch = ver[:epochIndex]
		ver = ver[epochIndex+1:]
	}

	name := filename[:verIndex]
	pkg := models.Package{
		Name:     name,
		Epoch:    epoch,
		Version:  ver,
		Release:  rel,
		Arch:     arch,
		Filename: filename,
	}
	return pkg, nil
}

// uniquePackages returns deduplicated []Package by Filename
// If Filename is the same, all other information is considered to be the same
func uniquePackages(pkgs []models.Package) []models.Package {
	tmp := make(map[string]models.Package)
	ret := []models.Package{}
	for _, pkg := range pkgs {
		tmp[pkg.Filename] = pkg
	}
	for _, v := range tmp {
		ret = append(ret, v)
	}
	return ret
}

func mergeUpdates(source map[string]*models.Updates, target map[string]*models.Updates) map[string]*models.Updates {
	for osVer, sourceUpdates := range source {
		if targetUpdates, ok := target[osVer]; ok {
			source[osVer].UpdateList = append(sourceUpdates.UpdateList, targetUpdates.UpdateList...)
		}
	}
	return source
}
