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
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestSGClassesGenerate(t *testing.T) {
	cnds, err := SCClassesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":         "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"storage.k8s.io/v1\\\",\\\"kind\\\":\\\"StorageClass\\\",\\\"metadata\\\":{\\\"annotations\\\":{\\\"storageclass.kubernetes.io/is-default-class\\\":\\\"true\\\"},\\\"name\\\":\\\"gp2\\\"},\\\"parameters\\\":{\\\"fsType\\\":\\\"ext4\\\",\\\"type\\\":\\\"gp2\\\"},\\\"provisioner\\\":\\\"kubernetes.io/aws-ebs\\\",\\\"volumeBindingMode\\\":\\\"WaitForFirstConsumer\\\"}\\n\",\"storageclass.kubernetes.io/is-default-class\":\"true\"}",
			"cluster_uid":         "e7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":  "1609173285",
			"name":                "gp2",
			"parameters":          "{\"fsType\":\"ext4\",\"type\":\"gp2\"}",
			"provisioner":         "kubernetes.io/aws-ebs",
			"reclaim_policy":      "Delete",
			"uid":                 "4dae2799-6576-403c-8644-7a2ad12b1fd7",
			"volume_binding_mode": "WaitForFirstConsumer",
		},
	}, cnds)
}
