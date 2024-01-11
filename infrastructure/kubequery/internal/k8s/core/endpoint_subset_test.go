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

func TestEndpointSubsetsGenerate(t *testing.T) {
	ess, err := EndpointSubsetsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"addresses":          "[{\"ip\":\"10.1.26.50\",\"nodeName\":\"seshu\",\"targetRef\":{\"kind\":\"Pod\",\"namespace\":\"default\",\"name\":\"jaeger-operator-5db4f9d996-pm7ld\",\"uid\":\"2271363b-ffc9-4f00-984c-e0a125ee2d7a\",\"resourceVersion\":\"451808\"}}]",
			"annotations":        "{\"endpoints.kubernetes.io/last-change-trigger-time\":\"2021-01-20T20:08:52-05:00\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191332",
			"labels":             "{\"name\":\"jaeger-operator\"}",
			"name":               "jaeger-operator",
			"namespace":          "default",
			"ports":              "[{\"name\":\"metrics\",\"port\":8383,\"protocol\":\"TCP\"}]",
			"uid":                "013741da-d7a5-4a2d-8f4b-792ac6a40dd3",
		},
	}, ess)
}
