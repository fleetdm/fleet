/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package apps

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/apps/v1"
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
	ds := &v1.DaemonSet{}
	loadTestResource("daemon_set_test.json", ds)
	d := &v1.Deployment{}
	loadTestResource("deployment_test.json", d)
	rs := &v1.ReplicaSet{}
	loadTestResource("replica_set_test.json", rs)
	ss := &v1.StatefulSet{}
	loadTestResource("stateful_set_test.json", ss)

	k8s.SetClient(fake.NewSimpleClientset(ds, d, rs, ss), types.UID("blah"), "")
}
