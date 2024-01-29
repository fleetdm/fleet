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

func TestNetworkPoliciesGenerate(t *testing.T) {
	nps, err := NetworkPoliciesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"networking.k8s.io/v1\\\",\\\"kind\\\":\\\"NetworkPolicy\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"test-network-policy\\\",\\\"namespace\\\":\\\"default\\\"},\\\"spec\\\":{\\\"egress\\\":[{\\\"ports\\\":[{\\\"port\\\":5978,\\\"protocol\\\":\\\"TCP\\\"}],\\\"to\\\":[{\\\"ipBlock\\\":{\\\"cidr\\\":\\\"10.0.0.0/24\\\"}}]}],\\\"ingress\\\":[{\\\"from\\\":[{\\\"ipBlock\\\":{\\\"cidr\\\":\\\"172.17.0.0/16\\\",\\\"except\\\":[\\\"172.17.1.0/24\\\"]}},{\\\"namespaceSelector\\\":{\\\"matchLabels\\\":{\\\"project\\\":\\\"myproject\\\"}}},{\\\"podSelector\\\":{\\\"matchLabels\\\":{\\\"role\\\":\\\"frontend\\\"}}}],\\\"ports\\\":[{\\\"port\\\":6379,\\\"protocol\\\":\\\"TCP\\\"}]}],\\\"podSelector\\\":{\\\"matchLabels\\\":{\\\"role\\\":\\\"db\\\"}},\\\"policyTypes\\\":[\\\"Ingress\\\",\\\"Egress\\\"]}}\\n\"}",
			"cluster_uid":        "c7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611328106",
			"from_to":            "[{\"ipBlock\":{\"cidr\":\"172.17.0.0/16\",\"except\":[\"172.17.1.0/24\"]}},{\"namespaceSelector\":{\"matchLabels\":{\"project\":\"myproject\"}}},{\"podSelector\":{\"matchLabels\":{\"role\":\"frontend\"}}}]",
			"name":               "test-network-policy",
			"namespace":          "default",
			"pod_selector":       "{\"matchLabels\":{\"role\":\"db\"}}",
			"policy_types":       "[\"Ingress\",\"Egress\"]",
			"ports":              "[{\"protocol\":\"TCP\",\"port\":6379}]",
			"type":               "ingress",
			"uid":                "ef70a000-9460-4098-9100-1d2b4bf608e1",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"networking.k8s.io/v1\\\",\\\"kind\\\":\\\"NetworkPolicy\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"test-network-policy\\\",\\\"namespace\\\":\\\"default\\\"},\\\"spec\\\":{\\\"egress\\\":[{\\\"ports\\\":[{\\\"port\\\":5978,\\\"protocol\\\":\\\"TCP\\\"}],\\\"to\\\":[{\\\"ipBlock\\\":{\\\"cidr\\\":\\\"10.0.0.0/24\\\"}}]}],\\\"ingress\\\":[{\\\"from\\\":[{\\\"ipBlock\\\":{\\\"cidr\\\":\\\"172.17.0.0/16\\\",\\\"except\\\":[\\\"172.17.1.0/24\\\"]}},{\\\"namespaceSelector\\\":{\\\"matchLabels\\\":{\\\"project\\\":\\\"myproject\\\"}}},{\\\"podSelector\\\":{\\\"matchLabels\\\":{\\\"role\\\":\\\"frontend\\\"}}}],\\\"ports\\\":[{\\\"port\\\":6379,\\\"protocol\\\":\\\"TCP\\\"}]}],\\\"podSelector\\\":{\\\"matchLabels\\\":{\\\"role\\\":\\\"db\\\"}},\\\"policyTypes\\\":[\\\"Ingress\\\",\\\"Egress\\\"]}}\\n\"}",
			"cluster_uid":        "c7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611328106",
			"from_to":            "[{\"ipBlock\":{\"cidr\":\"10.0.0.0/24\"}}]",
			"name":               "test-network-policy",
			"namespace":          "default",
			"pod_selector":       "{\"matchLabels\":{\"role\":\"db\"}}",
			"policy_types":       "[\"Ingress\",\"Egress\"]",
			"ports":              "[{\"protocol\":\"TCP\",\"port\":5978}]",
			"type":               "egress",
			"uid":                "ef70a000-9460-4098-9100-1d2b4bf608e1",
		},
	}, nps)
}
