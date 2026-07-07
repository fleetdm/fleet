package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUp_20260707140102(t *testing.T) {
	db := applyUpToPrev(t)

	assert.False(t, indexExists(db, "nano_enrollment_queue", "idx_neq_next_command"))

	applyNext(t, db)

	assert.True(t, indexExists(db, "nano_enrollment_queue", "idx_neq_next_command"))
}
