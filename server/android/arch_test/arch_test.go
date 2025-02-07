package arch_test

import (
	"container/list"
	"go/build"
	"regexp"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestPackageDependencies checks that android packages are not dependent on other packages
// to maintain decoupling and modularity.
func TestPackageDependencies(t *testing.T) {
	packageName := "github.com/fleetdm/fleet/v4/server/android..."
	NewPackageTest(t, packageName).
		WithIncludeRegex(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(
			"github.com/fleetdm/fleet/v4/server/service",
			"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql...",
			"github.com/fleetdm/fleet/v4/server/datastore/filesystem...",
			"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql...",
			"github.com/fleetdm/fleet/v4/server/datastore/mysql",
			"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations...",
			"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis...",
			"github.com/fleetdm/fleet/v4/server/datastore/redis...",
			"github.com/fleetdm/fleet/v4/server/datastore/s3...",
			// "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm",
		)
}

type PackageTest struct {
	t            TestingT
	pkgs         []string
	includeRegex *regexp.Regexp
}

type TestingT interface {
	Errorf(format string, args ...any)
}

func NewPackageTest(t TestingT, packageName ...string) *PackageTest {
	return &PackageTest{t: t, pkgs: packageName}
}

// WithIncludeRegex sets a regex to filter the packages to include in the dependency check.
// This significantly speeds up the dependency check by only importing the packages that match the regex.
func (pt *PackageTest) WithIncludeRegex(regex *regexp.Regexp) *PackageTest {
	pt.includeRegex = regex
	return pt
}

func (pt *PackageTest) ShouldNotDependOn(pkgs ...string) {
	expandedPackages := pt.expandPackages(pkgs)
	for dep := range pt.findDependencies(pt.pkgs) {
		if dep.isDependencyOn(expandedPackages) {
			pt.t.Errorf("Error: package dependency not allowed. Dependency chain:\n%s", dep)
		}
	}
}

type packageDependency struct {
	name   string
	parent *packageDependency
}

func (pd *packageDependency) String() string {
	result, _ := pd.chain()
	return result
}

func (pd *packageDependency) chain() (string, int) {
	name := pd.name

	if pd.parent == nil {
		return name + "\n", 1
	}

	pc, numberOfTabs := pd.parent.chain()

	return pc + strings.Repeat("\t", numberOfTabs) + name + "\n", numberOfTabs + 1
}

func (pd *packageDependency) isDependencyOn(pkgs []string) bool {
	if pd.parent == nil {
		return false
	}
	return slices.Contains(pkgs, pd.name)
}

func (pt PackageTest) findDependencies(pkgs []string) <-chan *packageDependency {
	c := make(chan *packageDependency)
	go func() {
		defer close(c)

		importCache := map[string]struct{}{}
		for _, p := range pt.expandPackages(pkgs) {
			pt.read(c, &packageDependency{name: p, parent: nil}, importCache)
		}
	}()
	return c
}

func (pt *PackageTest) read(pChan chan<- *packageDependency, topDependency *packageDependency, cache map[string]struct{}) {
	queue := list.New()
	queue.PushBack(topDependency)
	for queue.Len() > 0 {
		front := queue.Front()
		queue.Remove(front)
		dep, _ := (front.Value).(*packageDependency)

		if _, seen := cache[dep.name]; seen || dep.name == "C" {
			continue
		}

		if pt.includeRegex != nil && !pt.includeRegex.MatchString(dep.name) {
			continue
		}

		cache[dep.name] = struct{}{}
		pChan <- dep

		pkg, err := build.Default.Import(dep.name, ".", build.ImportMode(0))
		if err != nil {
			pt.t.Errorf("Error reading: %s", dep.name)
			continue
		}
		if pkg.Goroot {
			continue
		}

		for _, importPath := range pkg.Imports {
			queue.PushBack(&packageDependency{name: importPath, parent: dep})
		}

	}
}

func (pt PackageTest) expandPackages(pkgs []string) []string {
	if !needExpansion(pkgs) {
		return pkgs
	}

	loadedPkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName}, pkgs...)
	if err != nil {
		pt.t.Errorf("Error reading: %s, err: %s", pkgs, err)
		return nil
	}
	if len(loadedPkgs) == 0 {
		pt.t.Errorf("Error reading: %s, did not match any packages", pkgs)
		return nil

	}

	packagePaths := make([]string, 0, len(loadedPkgs))
	for _, p := range loadedPkgs {
		packagePaths = append(packagePaths, p.PkgPath)
	}
	return packagePaths
}

func needExpansion(packages []string) bool {
	return slices.ContainsFunc(packages, func(p string) bool {
		return strings.Contains(p, "...")
	})
}
