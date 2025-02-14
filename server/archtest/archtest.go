package archtest

import (
	"container/list"
	"go/build"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PackageTest is an architecture test to check package dependencies.
// It is used to ensure that packages do not depend on each other in a way that increases coupling and maintainability.
// Based on https://github.com/matthewmcnew/archtest
type PackageTest struct {
	t            TestingT
	pkgs         []string
	includeRegex *regexp.Regexp
	ignorePkgs   map[string]struct{}
	ignoreXTests map[string]struct{}
	withTests    bool
}

// PackageTest will ignore dependency on this package.
const thisPackage = "github.com/fleetdm/fleet/v4/server/archtest"

type TestingT interface {
	Errorf(format string, args ...any)
}

func NewPackageTest(t TestingT, packageName ...string) *PackageTest {
	return &PackageTest{t: t, pkgs: packageName}
}

// OnlyInclude sets a regex to filter the packages to include in the dependency check.
// This significantly speeds up the dependency check by only importing the packages that match the regex.
func (pt *PackageTest) OnlyInclude(regex *regexp.Regexp) *PackageTest {
	pt.includeRegex = regex
	return pt
}

func (pt *PackageTest) IgnorePackages(pkgs ...string) *PackageTest {
	if pt.ignorePkgs == nil {
		pt.ignorePkgs = make(map[string]struct{}, len(pkgs))
	}
	for _, p := range pt.expandPackages(pkgs) {
		pt.ignorePkgs[p] = struct{}{}
	}
	return pt
}

func (pt *PackageTest) IgnoreXTests(pkgs ...string) *PackageTest {
	if pt.ignoreXTests == nil {
		pt.ignoreXTests = make(map[string]struct{}, len(pkgs))
	}
	cleanPkgs := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		cleanPkgs = append(cleanPkgs, strings.TrimSuffix(p, "_test"))
	}
	for _, p := range pt.expandPackages(cleanPkgs) {
		pt.ignoreXTests[p] = struct{}{}
	}
	return pt
}

func (pt *PackageTest) WithTests() *PackageTest {
	pt.withTests = true
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
	xTest  bool
}

func (pd *packageDependency) String() string {
	result, _ := pd.chain()
	return result
}

func (pd *packageDependency) chain() (string, int) {
	name := pd.name
	if pd.xTest {
		name += "_test"
	}
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

// asXTest marks returns a copy of package dependency marked as external test.
func (pd packageDependency) asXTest() *packageDependency {
	pd.xTest = true
	return &pd
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

		if pt.skip(cache, dep) {
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

		if pt.withTests {
			for _, i := range pkg.TestImports {
				queue.PushBack(&packageDependency{name: i, parent: dep})
			}

			// XTestImports are packages with _test suffix that are in the same directory as the package.
			if _, ignore := pt.ignoreXTests[dep.name]; ignore {
				continue
			}
			for _, i := range pkg.XTestImports {
				queue.PushBack(&packageDependency{name: i, parent: dep.asXTest()})
			}
		}
	}
}

func (pt *PackageTest) skip(cache map[string]struct{}, dep *packageDependency) bool {
	if _, seen := cache[dep.name]; seen {
		return true
	}

	if _, ignore := pt.ignorePkgs[dep.name]; ignore || dep.name == "C" || dep.name == thisPackage {
		return true
	}

	if pt.includeRegex != nil && !pt.includeRegex.MatchString(dep.name) {
		return true
	}
	return false
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
