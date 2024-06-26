/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package rbac

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestClusterRoleBindingSubjectsGenerate(t *testing.T) {
	crbss, err := ClusterRoleBindingSubjectsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRoleBinding\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"kubernetes-dashboard\\\"},\\\"roleRef\\\":{\\\"apiGroup\\\":\\\"rbac.authorization.k8s.io\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"name\\\":\\\"kubernetes-dashboard\\\"},\\\"subjects\\\":[{\\\"kind\\\":\\\"ServiceAccount\\\",\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"}]}\\n\"}",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"name":               "kubernetes-dashboard",
			"role_api_group":     "rbac.authorization.k8s.io",
			"role_kind":          "ClusterRole",
			"role_name":          "kubernetes-dashboard",
			"subject_kind":       "ServiceAccount",
			"subject_name":       "kubernetes-dashboard",
			"subject_namespace":  "kube-system",
			"uid":                "7e3bf161-3a4e-495d-98a8-f71248d0ba36",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRoleBinding\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"nginx-ingress-microk8s\\\"},\\\"roleRef\\\":{\\\"apiGroup\\\":\\\"rbac.authorization.k8s.io\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"name\\\":\\\"nginx-ingress-microk8s-clusterrole\\\"},\\\"subjects\\\":[{\\\"kind\\\":\\\"ServiceAccount\\\",\\\"name\\\":\\\"nginx-ingress-microk8s-serviceaccount\\\",\\\"namespace\\\":\\\"ingress\\\"}]}\\n\"}",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191047",
			"name":               "nginx-ingress-microk8s",
			"role_api_group":     "rbac.authorization.k8s.io",
			"role_kind":          "ClusterRole",
			"role_name":          "nginx-ingress-microk8s-clusterrole",
			"subject_kind":       "ServiceAccount",
			"subject_name":       "nginx-ingress-microk8s-serviceaccount",
			"subject_namespace":  "ingress",
			"uid":                "aa9c6e0e-3dd4-4da3-936a-a6edea62c7b7",
		},
	}, crbss)
}
