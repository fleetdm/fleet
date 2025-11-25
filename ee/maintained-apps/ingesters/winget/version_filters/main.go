// ABOUTME: Version filters for winget package ingestion - filters out invalid version directories before sorting
// ABOUTME: Follows the same registry pattern as external_refs for app-specific version selection logic

package versionfilters

import "github.com/google/go-github/v37/github"

// FilterFunc is a function that filters version directories for a specific package.
// It receives a slice of GitHub repository contents and returns a filtered slice.
type FilterFunc func([]*github.RepositoryContent) []*github.RepositoryContent

// Funcs is a registry of version filter functions keyed by winget package identifier.
// Filters are applied before version sorting to remove invalid/legacy version directories.
var Funcs = map[string]FilterFunc{
	"Adobe.Acrobat.Pro": AdobeAcrobatVersionFilter,
}

// ApplyFilters applies the registered version filter for the given package identifier.
// If no filter is registered for the package, the original contents are returned unchanged.
func ApplyFilters(packageID string, contents []*github.RepositoryContent) []*github.RepositoryContent {
	if filter, ok := Funcs[packageID]; ok {
		return filter(contents)
	}
	return contents
}
