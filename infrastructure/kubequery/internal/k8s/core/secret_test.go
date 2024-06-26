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

func TestSecretsGenerate(t *testing.T) {
	ss, err := SecretsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"istio.io/service-account.name\":\"jaeger-operator\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191305",
			"name":               "istio.jaeger-operator",
			"namespace":          "default",
			"type":               "istio.io/key-and-cert",
			"uid":                "fb60f655-6b24-4f35-8e2d-17d7ca3ba7d4",
		},
	}, ss)
}
