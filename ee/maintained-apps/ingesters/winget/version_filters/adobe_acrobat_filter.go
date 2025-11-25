package versionfilters

import "github.com/google/go-github/v37/github"

// AdobeAcrobatVersionFilter removes the "2020" legacy edition directory from version selection.
//
// Background:
// Adobe maintains two product lines for Acrobat Pro:
// - DC 2020: Perpetual license edition (frozen at 2020.x versions, discontinued)
// - Continuous: Subscription-based edition with ongoing updates (25.x, 26.x, 27.x, etc.)
//
// The winget repository contains both "2020" and semantic versioned directories (e.g., "25.001.20844").
// SmartVerCmp incorrectly ranks "2020" as newer than "25.001.20844" because it compares
// "2020" vs "25" numerically. This filter removes the legacy "2020" directory before sorting.
//
// Note: Hardcoding "2020" exclusion is acceptable because:
// - It's the only non-semantic-version directory in the Adobe Acrobat Pro repo
// - Future versions will follow X.Y.Z pattern (26.x, 27.x, etc.)
// - Adobe discontinued the perpetual license model after DC 2020
// - All continuous track versions use semantic versioning
func AdobeAcrobatVersionFilter(contents []*github.RepositoryContent) []*github.RepositoryContent {
	filtered := make([]*github.RepositoryContent, 0, len(contents))
	for _, item := range contents {
		if item.GetName() != "2020" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
