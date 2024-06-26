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

func TestNamespacesGenerate(t *testing.T) {
	nss, err := NamespacesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1610476216",
			"name":               "default",
			"phase":              "Active",
			"uid":                "7b50dc9c-6149-4cac-a0d0-52bf0fa5356d",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"v1\\\",\\\"kind\\\":\\\"Namespace\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"ingress\\\"}}\\n\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191047",
			"name":               "ingress",
			"phase":              "Active",
			"uid":                "7653c4b9-3df2-493e-ae28-5e3a777f7e76",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"v1\\\",\\\"kind\\\":\\\"Namespace\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"istio-injection\\\":\\\"disabled\\\"},\\\"name\\\":\\\"istio-system\\\"}}\\n\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"istio-injection\":\"disabled\"}",
			"name":               "istio-system",
			"phase":              "Active",
			"uid":                "7f931f07-f8d0-4198-bf16-e459914e1866",
		},
		{
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1610476215",
			"name":               "kube-node-lease",
			"phase":              "Active",
			"uid":                "a8f303fd-0074-475f-935a-122cf8b6d1ad",
		},
		{
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1610476215",
			"name":               "kube-public",
			"phase":              "Active",
			"uid":                "6c719dfa-3de8-477b-a650-8bf9e2f12ee0",
		},
		{
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1610476215",
			"name":               "kube-system",
			"phase":              "Active",
			"uid":                "ebca5546-b939-4765-bf3d-869ac644ea0f",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"v1\\\",\\\"kind\\\":\\\"Namespace\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"monitoring\\\"}}\\n\"}",
			"cluster_uid":        "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191449",
			"name":               "monitoring",
			"phase":              "Active",
			"uid":                "afb98a87-39bb-4c8f-b0dd-8ea3683ba745",
		},
	}, nss)
}
