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

func TestLimitRangesGenerate(t *testing.T) {
	js, err := LimitRangesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid":             "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":      "0",
			"default":                 "{\"cpu\":\"3\"}",
			"default_request":         "{\"cpu\":\"2\"}",
			"labels":                  "{\"a\":\"b\"}",
			"max":                     "{\"cpu\":\"0\"}",
			"max_limit_request_ratio": "{\"cpu\":\"1\"}",
			"min":                     "{\"cpu\":\"4\"}",
			"name":                    "lr1",
			"namespace":               "n123",
			"type":                    "Container",
			"uid":                     "1234",
		},
	}, js)
}
