package profiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownCANames(t *testing.T) {
	assert.Equal(t, "", KnownCANames[int](nil))
	assert.Equal(t, "", KnownCANames(map[string]int{}))
	assert.Equal(t, "one", KnownCANames(map[string]int{"one": 1}))
	assert.Equal(t, "a,b,c", KnownCANames(map[string]struct{}{"c": {}, "a": {}, "b": {}}))
}
