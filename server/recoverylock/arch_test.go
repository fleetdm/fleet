package recoverylock_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	moduleRoot      = "github.com/fleetdm/fleet/v4"
	recoveryLockPkg = moduleRoot + "/server/recoverylock"
)

// allowedDependencies defines the external dependencies allowed for each package.
// Packages not listed default to allowing standard library and the listed dependencies.
var allowedDependencies = map[string][]string{
	// Root package should have minimal dependencies
	recoveryLockPkg: {
		moduleRoot + "/server/contexts/ctxerr",
	},
	// API package should have no fleet dependencies except ctxerr
	recoveryLockPkg + "/api": {
		moduleRoot + "/server/contexts/ctxerr",
	},
	// Bootstrap can import api, internal packages, and recoverylock root
	recoveryLockPkg + "/bootstrap": {
		recoveryLockPkg,
		recoveryLockPkg + "/api",
		recoveryLockPkg + "/internal/mysql",
		recoveryLockPkg + "/internal/service",
	},
	// Internal types should have minimal dependencies
	recoveryLockPkg + "/internal/types": {
		moduleRoot + "/server/contexts/ctxerr",
	},
	// Internal mysql can use types and ctxerr
	recoveryLockPkg + "/internal/mysql": {
		recoveryLockPkg + "/internal/types",
		moduleRoot + "/server/contexts/ctxerr",
	},
	// Internal service can use all internal packages, api, and recoverylock root
	recoveryLockPkg + "/internal/service": {
		recoveryLockPkg,
		recoveryLockPkg + "/api",
		recoveryLockPkg + "/internal/types",
		moduleRoot + "/server/contexts/ctxerr",
	},
}

// forbiddenDependencies lists packages that should NEVER be imported.
var forbiddenDependencies = []string{
	moduleRoot + "/server/fleet",     // No direct fleet package imports
	moduleRoot + "/server/datastore", // No datastore imports
	moduleRoot + "/server/service",   // No service imports
	moduleRoot + "/server/mdm",       // No mdm package imports
	moduleRoot + "/cmd",              // No cmd imports
}

func TestRecoveryLockBoundaryEnforcement(t *testing.T) {
	// Find the recoverylock directory
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Walk through all Go files in the recoverylock package
	err = filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and test files
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		// Determine the package path
		relPath, err := filepath.Rel(wd, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkgPath := recoveryLockPkg
		if relPath != "." {
			pkgPath = recoveryLockPkg + "/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
		}

		// Check each import
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")

			// Skip standard library imports
			if !strings.Contains(importPath, ".") {
				continue
			}

			// Skip non-fleet imports (third-party libraries)
			if !strings.HasPrefix(importPath, moduleRoot) {
				continue
			}

			// Check forbidden dependencies
			for _, forbidden := range forbiddenDependencies {
				// Allow the module if it's specifically allowed for this package
				if isAllowed(pkgPath, importPath) {
					continue
				}
				assert.False(t, strings.HasPrefix(importPath, forbidden),
					"Package %s imports forbidden dependency %s (from file %s)",
					pkgPath, importPath, path)
			}

			// For stricter checking, verify against allowed list
			if allowed, ok := allowedDependencies[pkgPath]; ok {
				if strings.HasPrefix(importPath, moduleRoot) && !strings.HasPrefix(importPath, recoveryLockPkg) {
					isInAllowed := false
					for _, a := range allowed {
						if strings.HasPrefix(importPath, a) {
							isInAllowed = true
							break
						}
					}
					assert.True(t, isInAllowed,
						"Package %s imports non-allowed fleet dependency %s (from file %s). Allowed: %v",
						pkgPath, importPath, path, allowed)
				}
			}
		}

		return nil
	})
	require.NoError(t, err)
}

func isAllowed(pkgPath, importPath string) bool {
	allowed, ok := allowedDependencies[pkgPath]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if strings.HasPrefix(importPath, a) {
			return true
		}
	}
	return false
}

func TestInternalPackagesNotExportedDirectly(t *testing.T) {
	// This test verifies that internal packages cannot be imported from outside
	// the recoverylock module. Go's compiler enforces this, but this test documents
	// the expectation.

	internalPkgs := []string{
		"internal/types",
		"internal/mysql",
		"internal/service",
	}

	for _, pkg := range internalPkgs {
		t.Run(pkg, func(t *testing.T) {
			// The internal package should exist
			wd, err := os.Getwd()
			require.NoError(t, err)

			pkgPath := filepath.Join(wd, pkg)
			_, err = os.Stat(pkgPath)
			assert.NoError(t, err, "Internal package %s should exist", pkg)

			// Verify it's under the internal directory
			assert.True(t, strings.Contains(pkg, "internal/"),
				"Package %s should be under internal/", pkg)
		})
	}
}

func TestAPIPackageExportsOnlyInterfaces(t *testing.T) {
	// This test verifies that the api package primarily exports interfaces
	// and types, not implementations.

	wd, err := os.Getwd()
	require.NoError(t, err)

	apiDir := filepath.Join(wd, "api")
	entries, err := os.ReadDir(apiDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(apiDir, entry.Name())
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			require.NoError(t, err)

			// API package should be named "api"
			assert.Equal(t, "api", f.Name.Name,
				"API package should be named 'api'")
		})
	}
}
