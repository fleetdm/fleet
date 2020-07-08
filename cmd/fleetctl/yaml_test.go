package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYaml(t *testing.T) {
	in := `
---
- Document
#---
--- Document2
---
Document3
`

	docs := splitYaml(in)
	require.Equal(t, 3, len(docs))
	assert.Equal(t, "- Document\n#---", docs[0])
	assert.Equal(t, "Document2", docs[1])
	assert.Equal(t, "Document3", docs[2])
}
