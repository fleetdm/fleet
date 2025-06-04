package archtest

import (
	"fmt"
	"strings"
	"testing"
)

const packagePrefix = "github.com/fleetdm/fleet/v4/server/archtest/test_files/"

func TestPackage_ShouldNotDependOn(t *testing.T) {

	t.Run("Succeeds on non dependencies", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"testpackage").
			ShouldNotDependOn(packagePrefix + "nodependency")

		assertNoError(t, mockT)
	})

	t.Run("Fails on dependencies", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"testpackage").
			ShouldNotDependOn(packagePrefix + "dependency")

		assertError(t, mockT,
			packagePrefix+"testpackage",
			packagePrefix+"dependency")
	})

	t.Run("Supports testing against packages in the go root", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"testpackage").
			ShouldNotDependOn("crypto")

		assertError(t, mockT,
			packagePrefix+"testpackage",
			"crypto")
	})

	t.Run("Fails on transative dependencies", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"testpackage").
			ShouldNotDependOn(packagePrefix + "transative")

		assertError(t, mockT,
			packagePrefix+"testpackage",
			packagePrefix+"dependency",
			packagePrefix+"transative")
	})

	t.Run("Supports multiple packages at once", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"dontdependonanything",
			packagePrefix+"testpackage").
			ShouldNotDependOn(packagePrefix+"nodependency",
				packagePrefix+"dependency")

		assertError(t, mockT,
			packagePrefix+"testpackage",
			packagePrefix+"dependency")
	})

	t.Run("Supports wildcard matching", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"...").
			ShouldNotDependOn(packagePrefix + "nodependency")

		assertNoError(t, mockT)

		NewPackageTest(mockT, packagePrefix+"testpackage/nested/...").
			ShouldNotDependOn(packagePrefix + "...")

		assertError(t, mockT, packagePrefix+"testpackage/nested/dep",
			packagePrefix+"nesteddependency")
	})

	t.Run("Supports checking imports in test files", func(t *testing.T) {
		mockT := new(testingT)

		NewPackageTest(mockT, packagePrefix+"testpackage/...").
			ShouldNotDependOn(packagePrefix + "testfiledeps/testonlydependency")

		assertNoError(t, mockT)

		NewPackageTest(mockT, packagePrefix+"testpackage/...").
			WithTests().
			ShouldNotDependOn(packagePrefix + "testfiledeps/testonlydependency")

		assertError(t, mockT,
			packagePrefix+"testpackage/nested/dep",
			packagePrefix+"testfiledeps/testonlydependency",
		)
	})

	t.Run("Supports checking imports from test packages", func(t *testing.T) {
		mockT := new(testingT)

		NewPackageTest(mockT, packagePrefix+"testpackage/...").
			ShouldNotDependOn(packagePrefix + "testfiledeps/testpkgdependency")

		assertNoError(t, mockT)

		NewPackageTest(mockT, packagePrefix+"testpackage/...").
			WithTests().
			ShouldNotDependOn(packagePrefix + "testfiledeps/testpkgdependency")

		assertError(t, mockT,
			packagePrefix+"testpackage/nested/dep_test",
			packagePrefix+"testfiledeps/testpkgdependency",
		)
	})

	t.Run("Supports Ignoring packages", func(t *testing.T) {
		mockT := new(testingT)

		NewPackageTest(mockT, packagePrefix+"testpackage/nested/dep").
			IgnorePackages(packagePrefix + "testpackage/nested/dep").
			ShouldNotDependOn(packagePrefix + "nesteddependency")

		assertNoError(t, mockT)
	})

	t.Run("Ignored packages ignore ignored transitive packages", func(t *testing.T) {
		mockT := new(testingT)

		NewPackageTest(mockT, packagePrefix+"testpackage").
			IgnorePackages("github.com/this/is/verifying/multiple/exclusions", packagePrefix+"...").
			IgnorePackages("github.com/this/is/verifying/chaining").
			ShouldNotDependOn(packagePrefix + "transative")

		assertNoError(t, mockT)
	})

	t.Run("Fails on packages that do not exist", func(t *testing.T) {
		mockT := new(testingT)
		NewPackageTest(mockT, packagePrefix+"dontexist/sorry").
			ShouldNotDependOn(packagePrefix + "dependency")

		assertError(t, mockT)

		mockT = new(testingT)
		NewPackageTest(mockT, "DONT__WORK").
			ShouldNotDependOn(packagePrefix + "dependency")

		assertError(t, mockT)

		mockT = new(testingT)
		NewPackageTest(mockT, packagePrefix+"dontexist/...").
			ShouldNotDependOn(packagePrefix + "dependency")

		assertError(t, mockT)
	})
}

func assertNoError(t *testing.T, mockT *testingT) {
	t.Helper()
	if mockT.errored() {
		t.Fatalf("archtest should not have failed but, %+v", mockT.message())
	}
}

func assertError(t *testing.T, mockT *testingT, dependencyTrace ...string) {
	t.Helper()
	if !mockT.errored() {
		t.Fatal("archtest did not fail on dependency")
	}

	if dependencyTrace == nil {
		return
	}

	s := strings.Builder{}
	s.WriteString("Error: package dependency not allowed. Dependency chain:\n")
	for i, v := range dependencyTrace {
		s.WriteString(strings.Repeat("\t", i))
		s.WriteString(v + "\n")
	}

	if mockT.message() != s.String() {
		t.Errorf("expected %s got error message: %s", s.String(), mockT.message())
	}
}

type testingT struct {
	errors [][]interface{}
}

func (t *testingT) Errorf(format string, args ...any) {
	t.errors = append(t.errors, append([]interface{}{format}, args...))
}

func (t testingT) errored() bool {
	return len(t.errors) != 0
}

func (t *testingT) message() string {
	if len(t.errors[0]) == 1 {
		return t.errors[0][0].(string)
	}
	return fmt.Sprintf(t.errors[0][0].(string), t.errors[0][1:]...)
}
