package common_mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcessSimple(t *testing.T) {
	payloads := []int{1, 2, 3, 4, 5}
	executeBatch := func(payloadsInThisBatch []int) error {
		t.Fatal("executeBatch should not be called")
		return nil
	}

	// No payloads
	err := BatchProcessSimple(nil, 10, executeBatch)
	assert.NoError(t, err)

	// No batch size
	err = BatchProcessSimple(payloads, 0, executeBatch)
	assert.NoError(t, err)

	// No executeBatch
	err = BatchProcessSimple(payloads, 10, nil)
	assert.NoError(t, err)

	// Large batch size -- all payloads executed in one batch
	executeBatch = func(payloadsInThisBatch []int) error {
		assert.Equal(t, payloads, payloadsInThisBatch)
		return nil
	}
	err = BatchProcessSimple(payloads, 10, executeBatch)
	assert.NoError(t, err)

	// Small batch size
	numCalls := 0
	executeBatch = func(payloadsInThisBatch []int) error {
		numCalls++
		switch numCalls {
		case 1:
			assert.Equal(t, []int{1, 2, 3}, payloadsInThisBatch)
		case 2:
			assert.Equal(t, []int{4, 5}, payloadsInThisBatch)
		default:
			t.Errorf("Unexpected number of calls to executeBatch: %d", numCalls)
		}
		return nil
	}
	err = BatchProcessSimple(payloads, 3, executeBatch)
	assert.NoError(t, err)
}
