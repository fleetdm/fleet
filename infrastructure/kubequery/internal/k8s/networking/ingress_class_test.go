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

func TestIngressClassesGenerate(t *testing.T) {
	igcs, err := IngressClassesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"ingressclass.kubernetes.io/is-default-class\":\"true\",\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"networking.k8s.io/v1\\\",\\\"kind\\\":\\\"IngressClass\\\",\\\"metadata\\\":{\\\"annotations\\\":{\\\"ingressclass.kubernetes.io/is-default-class\\\":\\\"true\\\"},\\\"name\\\":\\\"public\\\"},\\\"spec\\\":{\\\"controller\\\":\\\"k8s.io/ingress-nginx\\\"}}\\n\"}",
			"cluster_uid":        "c7fd8e77-93de-4742-9037-5db9a01e966a",
			"controller":         "k8s.io/ingress-nginx",
			"creation_timestamp": "1611191047",
			"name":               "public",
			"uid":                "dab8c076-3158-4a4a-8ee4-5632990ce074",
		},
	}, igcs)
}
