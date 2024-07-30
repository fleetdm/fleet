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
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/policy/v1"
	v1beta1 "k8s.io/api/policy/v1beta1"
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
	pdb := &v1.PodDisruptionBudget{}
	loadTestResource("pod_disruption_budget_test.json", pdb)
	psp := &v1beta1.PodSecurityPolicy{}
	loadTestResource("pod_security_policy_test.json", psp)

	k8s.SetClient(fake.NewSimpleClientset(pdb, psp), types.UID("b7fd8e77-93de-4742-9037-5db9a01e966a"), "")
}
