package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemovedPaths(t *testing.T) {
	t.Parallel()

	// removedPaths panics when a path has no removednames entry.
	assert.Len(t, removedPaths(), len(removedPathTemplates))

	for _, alias := range deprecatedPathAliases {
		for _, deprecated := range alias.DeprecatedPaths {
			assert.NotContains(t, removedPathTemplates, deprecated, "a removed path must not also be served as an alias of %s", alias.PrimaryPath)
		}
	}
}
