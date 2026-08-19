package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUp_20260527215817(t *testing.T) {
	db := applyUpToPrev(t)

	assert.False(t, indexExists(db, "host_certificates", "idx_host_certs_origin_deleted"))

	applyNext(t, db)

	assert.True(t, indexExists(db, "host_certificates", "idx_host_certs_origin_deleted"))
}
