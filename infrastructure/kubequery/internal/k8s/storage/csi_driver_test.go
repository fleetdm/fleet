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

func TestCSIDriversGenerate(t *testing.T) {
	cds, err := CSIDriversGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":            "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"storage.k8s.io/v1beta1\\\",\\\"kind\\\":\\\"CSIDriver\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"name\\\":\\\"efs.csi.aws.com\\\"},\\\"spec\\\":{\\\"attachRequired\\\":false}}\\n\"}",
			"attach_required":        "0",
			"cluster_uid":            "e7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":     "1609173285",
			"name":                   "efs.csi.aws.com",
			"pod_info_on_mount":      "0",
			"uid":                    "35613d4e-4f94-416c-bdad-88660302ce99",
			"volume_lifecycle_modes": "[\"Persistent\"]",
		},
	}, cds)
}
