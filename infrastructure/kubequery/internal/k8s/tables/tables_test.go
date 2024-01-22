/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTables(t *testing.T) {
	ts := GetTables()
	assert.NotNil(t, ts, "Invalid tables")
	assert.True(t, len(ts) > 40, "Invalid tables count")
}
