package mysql

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchHostnamesSmall(t *testing.T) {
	small := []string{"foo", "bar", "baz"}
	batched := batchHostnames(small)
	require.Equal(t, 1, len(batched))
	assert.Equal(t, small, batched[0])
}

func TestBatchHostnamesLarge(t *testing.T) {
	large := []string{}
	for i := 0; i < 230000; i++ {
		large = append(large, strconv.Itoa(i))
	}
	batched := batchHostnames(large)
	require.Equal(t, 5, len(batched))
	assert.Equal(t, large[:50000], batched[0])
	assert.Equal(t, large[50000:100000], batched[1])
	assert.Equal(t, large[100000:150000], batched[2])
	assert.Equal(t, large[150000:200000], batched[3])
	assert.Equal(t, large[200000:230000], batched[4])
}
