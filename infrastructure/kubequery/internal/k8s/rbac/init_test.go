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
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func loadTestResource(name string, v interface{}) {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

func init() {
	rs := &v1.Role{}
	loadTestResource("role_policy_rule_test.json", rs)
	rbs := &v1.RoleBinding{}
	loadTestResource("role_binding_subject_test.json", rbs)
	crs := &v1.ClusterRoleList{}
	loadTestResource("cluster_role_policy_rule_test.json", crs)
	crbs := &v1.ClusterRoleBindingList{}
	loadTestResource("cluster_role_binding_subject_test.json", crbs)

	k8s.SetClient(fake.NewSimpleClientset(rs, rbs, crs, crbs), types.UID("a7fd8e77-93de-4742-9037-5db9a01e966a"), "")
}
