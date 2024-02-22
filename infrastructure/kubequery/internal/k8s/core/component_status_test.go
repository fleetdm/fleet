/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package core

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestComponentStatusesGenerate(t *testing.T) {
	css, err := ComponentStatusesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid": "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"name":        "controller-manager",
			"message":     "ok",
			"status":      "True",
			"type":        "Healthy",
		},
		{
			"cluster_uid": "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"name":        "etcd-0",
			"message":     "{\"health\":\"true\"}",
			"status":      "True",
			"type":        "Healthy",
		},
		{
			"cluster_uid": "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"name":        "scheduler",
			"message":     "ok",
			"status":      "True",
			"type":        "Healthy",
		},
	}, css)
}
