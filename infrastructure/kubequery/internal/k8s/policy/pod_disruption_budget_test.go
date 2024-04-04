/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package policy

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestPodDisruptionBudgetsGenerate(t *testing.T) {
	pdbs, err := PodDisruptionBudgetsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":         "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"policy/v1\\\",\\\"kind\\\":\\\"PodDisruptionBudget\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"policy\\\",\\\"chart\\\":\\\"mixer\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"istio\\\":\\\"mixer\\\",\\\"istio-mixer-type\\\":\\\"policy\\\",\\\"release\\\":\\\"istio\\\",\\\"version\\\":\\\"1.5.1\\\"},\\\"name\\\":\\\"istio-policy\\\",\\\"namespace\\\":\\\"istio-system\\\"},\\\"spec\\\":{\\\"minAvailable\\\":1,\\\"selector\\\":{\\\"matchLabels\\\":{\\\"app\\\":\\\"policy\\\",\\\"istio\\\":\\\"mixer\\\",\\\"istio-mixer-type\\\":\\\"policy\\\",\\\"release\\\":\\\"istio\\\"}}}}\\n\"}",
			"cluster_uid":         "b7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":  "1611191143",
			"current_healthy":     "1",
			"desired_healthy":     "1",
			"disruptions_allowed": "0",
			"expected_pods":       "1",
			"labels":              "{\"app\":\"policy\",\"chart\":\"mixer\",\"heritage\":\"Tiller\",\"istio\":\"mixer\",\"istio-mixer-type\":\"policy\",\"release\":\"istio\",\"version\":\"1.5.1\"}",
			"min_available":       "1",
			"name":                "istio-policy",
			"namespace":           "istio-system",
			"observed_generation": "1",
			"selector":            "{\"matchLabels\":{\"app\":\"policy\",\"istio\":\"mixer\",\"istio-mixer-type\":\"policy\",\"release\":\"istio\"}}",
			"uid":                 "77dc4487-d95d-40a9-8fdb-f3bbe334c4e3",
		},
	}, pdbs)
}
