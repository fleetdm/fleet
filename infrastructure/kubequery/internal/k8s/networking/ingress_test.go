/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package networking

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestIngressesGenerate(t *testing.T) {
	igs, err := IngressesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid":        "c7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191344",
			"default_backend":    "{\"service\":{\"name\":\"simplest-query\",\"port\":{\"number\":16686}}}",
			"labels":             "{\"app\":\"jaeger\",\"app.kubernetes.io/component\":\"query-ingress\",\"app.kubernetes.io/instance\":\"simplest\",\"app.kubernetes.io/managed-by\":\"jaeger-operator\",\"app.kubernetes.io/name\":\"simplest-query\",\"app.kubernetes.io/part-of\":\"jaeger\"}",
			"load_balancer":      "{}",
			"name":               "simplest-query",
			"namespace":          "default",
			"uid":                "0cdc9181-0cb1-43bd-97b4-e31c864a13e2",
		},
	}, igs)
}
