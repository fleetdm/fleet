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
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestReplicaSetsGenerate(t *testing.T) {
	rss, err := ReplicaSetsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":                      "{\"deployment.kubernetes.io/desired-replicas\":\"1\",\"deployment.kubernetes.io/max-replicas\":\"2\",\"deployment.kubernetes.io/revision\":\"1\"}",
			"available_replicas":               "1",
			"cluster_uid":                      "blah",
			"creation_timestamp":               "1611191304",
			"dns_policy":                       "ClusterFirst",
			"fully_labeled_replicas":           "1",
			"host_ipc":                         "0",
			"host_network":                     "0",
			"host_pid":                         "0",
			"labels":                           "{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}",
			"min_ready_seconds":                "0",
			"name":                             "jaeger-operator-5db4f9d996",
			"namespace":                        "default",
			"observed_generation":              "1",
			"ready_replicas":                   "1",
			"replica_set_replicas":             "1",
			"replicas":                         "1",
			"restart_policy":                   "Always",
			"scheduler_name":                   "default-scheduler",
			"selector":                         "{\"matchLabels\":{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}}",
			"service_account_name":             "jaeger-operator",
			"termination_grace_period_seconds": "30",
			"uid":                              "2efeb411-ff99-434b-a5a2-4e06c2b0afaa",
		},
	}, rss)
}

func TestReplicaSetContainersGenerate(t *testing.T) {
	rss, err := ReplicaSetContainersGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":                "{\"deployment.kubernetes.io/desired-replicas\":\"1\",\"deployment.kubernetes.io/max-replicas\":\"2\",\"deployment.kubernetes.io/revision\":\"1\"}",
			"args":                       "[\"start\"]",
			"cluster_uid":                "blah",
			"container_type":             "container",
			"creation_timestamp":         "1611191304",
			"env":                        "[{\"name\":\"WATCH_NAMESPACE\"},{\"name\":\"POD_NAME\",\"valueFrom\":{\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"metadata.name\"}}},{\"name\":\"POD_NAMESPACE\",\"valueFrom\":{\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"metadata.namespace\"}}},{\"name\":\"OPERATOR_NAME\",\"value\":\"jaeger-operator\"}]",
			"image":                      "jaegertracing/jaeger-operator:1.14.0",
			"image_pull_policy":          "Always",
			"labels":                     "{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}",
			"name":                       "jaeger-operator",
			"namespace":                  "default",
			"ports":                      "[{\"name\":\"metrics\",\"containerPort\":8383,\"protocol\":\"TCP\"}]",
			"replica_set_name":           "jaeger-operator-5db4f9d996",
			"stdin":                      "0",
			"stdin_once":                 "0",
			"termination_message_path":   "/dev/termination-log",
			"termination_message_policy": "File",
			"tty":                        "0",
			"uid":                        "a9c84883-3d97-5b99-8b20-9fcd5e626a02",
		},
	}, rss)
}

func TestReplicaSetVolumesGenerate(t *testing.T) {
	rss, err := ReplicaSetVolumesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{}, rss)
}
