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
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestServicesGenerate(t *testing.T) {
	ss, err := ServicesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_ip":                  "10.152.183.187",
			"cluster_ips":                 "[\"10.152.183.187\"]",
			"cluster_uid":                 "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":          "1611191332",
			"health_check_node_port":      "0",
			"labels":                      "{\"name\":\"jaeger-operator\"}",
			"load_balancer":               "{}",
			"name":                        "jaeger-operator",
			"namespace":                   "default",
			"ports":                       "[{\"name\":\"metrics\",\"protocol\":\"TCP\",\"port\":8383,\"targetPort\":8383}]",
			"publish_not_ready_addresses": "0",
			"selector":                    "{\"name\":\"jaeger-operator\"}",
			"session_affinity":            "None",
			"type":                        "ClusterIP",
			"uid":                         "d8dfda88-e2c5-479e-bb2d-d0964805a925",
		},
	}, ss)
}
