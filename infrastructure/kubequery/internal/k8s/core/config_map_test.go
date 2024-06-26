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

func TestConfigMapsGenerate(t *testing.T) {
	cms, err := ConfigMapsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191331",
			"name":               "jaeger-operator-lock",
			"namespace":          "default",
			"uid":                "eec6944c-5c13-4e30-8326-1a82e1962e4d",
		},
	}, cms)
}
