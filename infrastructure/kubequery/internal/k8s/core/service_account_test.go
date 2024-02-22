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

func TestServiceAccountsGenerate(t *testing.T) {
	sas, err := ServiceAccountsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"v1\\\",\\\"kind\\\":\\\"ServiceAccount\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"istio-ingressgateway\\\",\\\"chart\\\":\\\"gateways\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"istio-ingressgateway-service-account\\\",\\\"namespace\\\":\\\"istio-system\\\"}}\\n\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"istio-ingressgateway\",\"chart\":\"gateways\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "istio-ingressgateway-service-account",
			"namespace":          "istio-system",
			"secrets":            "[{\"name\":\"istio-ingressgateway-service-account-token-zmk8b\"}]",
			"uid":                "de09c78a-ea26-42ff-82d5-2f7d3f24a8d1",
		},
	}, sas)
}
