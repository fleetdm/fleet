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
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	lr := &v1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lr1",
			Namespace: "n123",
			UID:       types.UID("1234"),
			Labels:    map[string]string{"a": "b"},
		},
		Spec: v1.LimitRangeSpec{
			Limits: []v1.LimitRangeItem{
				{
					Type:                 v1.LimitTypeContainer,
					Max:                  v1.ResourceList{v1.ResourceCPU: resource.MustParse("0")},
					Min:                  v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
					Default:              v1.ResourceList{v1.ResourceCPU: resource.MustParse("3")},
					DefaultRequest:       v1.ResourceList{v1.ResourceCPU: resource.MustParse("2")},
					MaxLimitRequestRatio: v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
				},
			},
		},
	}

	csl := &v1.ComponentStatusList{}
	loadTestResource("component_status_test.json", csl)
	cm := &v1.ConfigMap{}
	loadTestResource("config_map_test.json", cm)
	ep := &v1.Endpoints{}
	loadTestResource("endpoint_subset_test.json", ep)
	ns := &v1.NamespaceList{}
	loadTestResource("namespaces_test.json", ns)
	node := &v1.Node{}
	loadTestResource("node_test.json", node)
	pod := &v1.Pod{}
	loadTestResource("pod_test.json", pod)
	secret := &v1.Secret{}
	loadTestResource("secret_test.json", secret)
	sa := &v1.ServiceAccount{}
	loadTestResource("service_account_test.json", sa)
	services := &v1.Service{}
	loadTestResource("services_test.json", services)

	k8s.SetClient(fake.NewSimpleClientset(lr, csl, cm, ep, ns, node, pod, secret, sa, services),
		types.UID("d7fd8e77-93de-4742-9037-5db9a01e966a"), "")
}
