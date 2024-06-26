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

func TestPodsGenerate(t *testing.T) {
	ps, err := PodsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":                      "{\"cni.projectcalico.org/podIP\":\"10.1.26.50/32\",\"cni.projectcalico.org/podIPs\":\"10.1.26.50/32\"}",
			"cluster_uid":                      "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"conditions":                       "[{\"type\":\"Initialized\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2021-01-21T01:08:25Z\"},{\"type\":\"Ready\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2021-01-21T01:08:52Z\"},{\"type\":\"ContainersReady\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2021-01-21T01:08:52Z\"},{\"type\":\"PodScheduled\",\"status\":\"True\",\"lastProbeTime\":null,\"lastTransitionTime\":\"2021-01-21T01:08:25Z\"}]",
			"container_statuses":               "[{\"name\":\"jaeger-operator\",\"state\":{\"running\":{\"startedAt\":\"2021-01-21T01:08:51Z\"}},\"lastState\":{\"terminated\":{\"exitCode\":1,\"reason\":\"Error\",\"startedAt\":\"2021-01-21T01:08:36Z\",\"finishedAt\":\"2021-01-21T01:08:36Z\",\"containerID\":\"containerd://d4c9607e13f2bd2eec99f5261693557963a1380cfe6aceda23b9e3d3d195962f\"}},\"ready\":true,\"restartCount\":2,\"image\":\"docker.io/jaegertracing/jaeger-operator:1.14.0\",\"imageID\":\"docker.io/jaegertracing/jaeger-operator@sha256:5a3198179f7972028a29dd7fbf71ac7a21e0dbf46c85e8cc2c37e3b6a5ee26a4\",\"containerID\":\"containerd://4a8e3f149f24fb5d4429f4a38e86097e1aec3b6b174bb382a44c6706ad4406e1\",\"started\":true}]",
			"creation_timestamp":               "1611191305",
			"dns_policy":                       "ClusterFirst",
			"enable_service_links":             "1",
			"host_ip":                          "192.168.0.28",
			"host_ipc":                         "0",
			"host_network":                     "0",
			"host_pid":                         "0",
			"labels":                           "{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}",
			"name":                             "jaeger-operator-5db4f9d996-pm7ld",
			"namespace":                        "default",
			"node_name":                        "seshu",
			"phase":                            "Running",
			"pod_ip":                           "10.1.26.50",
			"pod_ips":                          "[{\"ip\":\"10.1.26.50\"}]",
			"preemption_policy":                "PreemptLowerPriority",
			"priority":                         "0",
			"qos_class":                        "BestEffort",
			"restart_policy":                   "Always",
			"scheduler_name":                   "default-scheduler",
			"service_account_name":             "jaeger-operator",
			"start_time":                       "1611191305",
			"termination_grace_period_seconds": "30",
			"tolerations":                      "[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":300}]",
			"uid":                              "2271363b-ffc9-4f00-984c-e0a125ee2d7a",
		},
	}, ps)
}

func TestPodContainersGenerate(t *testing.T) {
	pcs, err := PodContainersGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":                "{\"cni.projectcalico.org/podIP\":\"10.1.26.50/32\",\"cni.projectcalico.org/podIPs\":\"10.1.26.50/32\"}",
			"args":                       "[\"start\"]",
			"cluster_uid":                "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"container_id":               "4a8e3f149f24fb5d4429f4a38e86097e1aec3b6b174bb382a44c6706ad4406e1",
			"container_type":             "container",
			"creation_timestamp":         "1611191305",
			"env":                        "[{\"name\":\"WATCH_NAMESPACE\"},{\"name\":\"POD_NAME\",\"valueFrom\":{\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"metadata.name\"}}},{\"name\":\"POD_NAMESPACE\",\"valueFrom\":{\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"metadata.namespace\"}}},{\"name\":\"OPERATOR_NAME\",\"value\":\"jaeger-operator\"}]",
			"image":                      "jaegertracing/jaeger-operator:1.14.0",
			"image_repo":                 "docker.io/jaegertracing/jaeger-operator",
			"image_id":                   "5a3198179f7972028a29dd7fbf71ac7a21e0dbf46c85e8cc2c37e3b6a5ee26a4",
			"image_pull_policy":          "Always",
			"labels":                     "{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}",
			"last_termination_state":     "{\"terminated\":{\"exitCode\":1,\"reason\":\"Error\",\"startedAt\":\"2021-01-21T01:08:36Z\",\"finishedAt\":\"2021-01-21T01:08:36Z\",\"containerID\":\"containerd://d4c9607e13f2bd2eec99f5261693557963a1380cfe6aceda23b9e3d3d195962f\"}}",
			"name":                       "jaeger-operator",
			"namespace":                  "default",
			"pod_name":                   "jaeger-operator-5db4f9d996-pm7ld",
			"ports":                      "[{\"name\":\"metrics\",\"containerPort\":8383,\"protocol\":\"TCP\"}]",
			"ready":                      "1",
			"restart_count":              "2",
			"started":                    "1",
			"state":                      "{\"running\":{\"startedAt\":\"2021-01-21T01:08:51Z\"}}",
			"stdin":                      "0",
			"stdin_once":                 "0",
			"termination_message_path":   "/dev/termination-log",
			"termination_message_policy": "File",
			"tty":                        "0",
			"uid":                        "2e7d1ce3-8546-5b73-beb8-46c109f37668",
			"volume_mounts":              "[{\"name\":\"jaeger-operator-token-c94jx\",\"readOnly\":true,\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\"}]",
		},
	}, pcs)
}

func TestPodVolumesGenerate(t *testing.T) {
	pcs, err := PodVolumesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":                       "{\"cni.projectcalico.org/podIP\":\"10.1.26.50/32\",\"cni.projectcalico.org/podIPs\":\"10.1.26.50/32\"}",
			"aws_elastic_block_store_partition": "0",
			"cluster_uid":                       "d7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":                "1611191305",
			"gce_persistent_disk_partition":     "0",
			"iscsi_discovery_chap_auth":         "0",
			"iscsi_lun":                         "0",
			"iscsi_session_chap_auth":           "0",
			"labels":                            "{\"name\":\"jaeger-operator\",\"pod-template-hash\":\"5db4f9d996\"}",
			"name":                              "jaeger-operator-token-c94jx",
			"namespace":                         "default",
			"pod_name":                          "jaeger-operator-5db4f9d996-pm7ld",
			"scale_iossl_enabled":               "0",
			"secret_default_mode":               "420",
			"secret_name":                       "jaeger-operator-token-c94jx",
			"uid":                               "2271363b-ffc9-4f00-984c-e0a125ee2d7a",
			"volume_type":                       "secret",
		},
	}, pcs)
}
