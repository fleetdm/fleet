package inmem

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestApplyLimitOffset(t *testing.T) {
	im := Datastore{}
	data := []int{}

	// should work with empty
	low, high := im.getLimitOffsetSliceBounds(fleet.ListOptions{}, len(data))
	result := data[low:high]
	assert.Len(t, result, 0)
	low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{Page: 1, PerPage: 20}, len(data))
	result = data[low:high]
	assert.Len(t, result, 0)

	// insert some data
	for i := 0; i < 100; i++ {
		data = append(data, i)
	}

	// unlimited
	low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{}, len(data))
	result = data[low:high]
	assert.Len(t, result, 100)
	assert.Equal(t, data, result)

	// reasonable limit page 0
	low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{PerPage: 20}, len(data))
	result = data[low:high]
	assert.Len(t, result, 20)
	assert.Equal(t, data[:20], result)

	// too many per page
	low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{PerPage: 200}, len(data))
	result = data[low:high]
	assert.Len(t, result, 100)
	assert.Equal(t, data, result)

	// offset should be past end (zero results)
	low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{Page: 1, PerPage: 200}, len(data))
	result = data[low:high]
	assert.Len(t, result, 0)

	// all pages appended should equal the original data
	result = []int{}
	for i := 0; i < 5; i++ { // 5 used intentionally
		low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{Page: uint(i), PerPage: 25}, len(data))
		result = append(result, data[low:high]...)
	}
	assert.Len(t, result, 100)
	assert.Equal(t, data, result)

	// again with different params
	result = []int{}
	for i := 0; i < 100; i++ { // 5 used intentionally
		low, high = im.getLimitOffsetSliceBounds(fleet.ListOptions{Page: uint(i), PerPage: 1}, len(data))
		result = append(result, data[low:high]...)
	}
	assert.Len(t, result, 100)
	assert.Equal(t, data, result)

}
