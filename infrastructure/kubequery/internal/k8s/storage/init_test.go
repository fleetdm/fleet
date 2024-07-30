/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package storage

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/storage/v1"
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
	cd := &v1.CSIDriver{}
	loadTestResource("csi_driver_test.json", cd)
	cnd := &v1.CSINodeList{}
	loadTestResource("csi_node_driver_test.json", cnd)
	sc := &v1.StorageClass{}
	loadTestResource("storage_class_test.json", sc)

	k8s.SetClient(fake.NewSimpleClientset(cd, cnd, sc), types.UID("e7fd8e77-93de-4742-9037-5db9a01e966a"), "")
}
