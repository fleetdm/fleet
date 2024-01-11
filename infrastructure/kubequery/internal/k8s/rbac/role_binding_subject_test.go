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

func TestRoleBindingSubjectsGenerate(t *testing.T) {
	rbss, err := RoleBindingSubjectsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"RoleBinding\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"},\\\"roleRef\\\":{\\\"apiGroup\\\":\\\"rbac.authorization.k8s.io\\\",\\\"kind\\\":\\\"Role\\\",\\\"name\\\":\\\"kubernetes-dashboard\\\"},\\\"subjects\\\":[{\\\"kind\\\":\\\"ServiceAccount\\\",\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"}]}\\n\"}",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"namespace":          "kube-system",
			"role_kind":          "Role",
			"role_name":          "kubernetes-dashboard",
			"subject_kind":       "ServiceAccount",
			"subject_name":       "kubernetes-dashboard",
			"subject_namespace":  "kube-system",
			"uid":                "216b24d7-0611-4cb9-991b-fad53856241d",
		},
	}, rbss)
}
