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

func TestStatefulSetsGenerate(t *testing.T) {
	sss, err := StatefulSetsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"available_replicas":               "0",
			"cluster_uid":                      "blah",
			"collision_count":                  "0",
			"creation_timestamp":               "1611191592",
			"current_replicas":                 "1",
			"current_revision":                 "alertmanager-main-6674894c9d",
			"dns_policy":                       "ClusterFirst",
			"fs_group":                         "2000",
			"host_ipc":                         "0",
			"host_network":                     "0",
			"host_pid":                         "0",
			"labels":                           "{\"alertmanager\":\"main\"}",
			"name":                             "alertmanager-main",
			"namespace":                        "monitoring",
			"node_selector":                    "{\"kubernetes.io/os\":\"linux\"}",
			"observed_generation":              "1",
			"pod_management_policy":            "Parallel",
			"ready_replicas":                   "1",
			"replicas":                         "1",
			"restart_policy":                   "Always",
			"revision_history_limit":           "10",
			"run_as_non_root":                  "1",
			"run_as_user":                      "1000",
			"scheduler_name":                   "default-scheduler",
			"selector":                         "{\"matchLabels\":{\"alertmanager\":\"main\",\"app\":\"alertmanager\"}}",
			"service_account_name":             "alertmanager-main",
			"service_name":                     "alertmanager-operated",
			"stateful_set_replicas":            "1",
			"termination_grace_period_seconds": "120",
			"uid":                              "3c488e7e-420c-4515-b377-5dc3ee082744",
			"update_revision":                  "alertmanager-main-6674894c9d",
			"update_strategy":                  "{\"type\":\"RollingUpdate\"}",
			"updated_replicas":                 "1",
		},
	}, sss)
}

func TestStatefulSetContainersGenerate(t *testing.T) {
	sss, err := StatefulSetContainersGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"args":                       "[\"--config.file=/etc/alertmanager/config/alertmanager.yaml\",\"--storage.path=/alertmanager\",\"--data.retention=120h\",\"--cluster.listen-address=\",\"--web.listen-address=:9093\",\"--web.route-prefix=/\",\"--cluster.peer=alertmanager-main-0.alertmanager-operated:9094\"]",
			"cluster_uid":                "blah",
			"container_type":             "container",
			"creation_timestamp":         "1611191592",
			"env":                        "[{\"name\":\"POD_IP\",\"valueFrom\":{\"fieldRef\":{\"apiVersion\":\"v1\",\"fieldPath\":\"status.podIP\"}}}]",
			"image":                      "quay.io/prometheus/alertmanager:v0.21.0",
			"image_pull_policy":          "IfNotPresent",
			"labels":                     "{\"alertmanager\":\"main\"}",
			"liveness_probe":             "{\"httpGet\":{\"path\":\"/-/healthy\",\"port\":\"web\",\"scheme\":\"HTTP\"},\"timeoutSeconds\":3,\"periodSeconds\":10,\"successThreshold\":1,\"failureThreshold\":10}",
			"name":                       "alertmanager",
			"namespace":                  "monitoring",
			"ports":                      "[{\"name\":\"web\",\"containerPort\":9093,\"protocol\":\"TCP\"},{\"name\":\"mesh-tcp\",\"containerPort\":9094,\"protocol\":\"TCP\"},{\"name\":\"mesh-udp\",\"containerPort\":9094,\"protocol\":\"UDP\"}]",
			"readiness_probe":            "{\"httpGet\":{\"path\":\"/-/ready\",\"port\":\"web\",\"scheme\":\"HTTP\"},\"initialDelaySeconds\":3,\"timeoutSeconds\":3,\"periodSeconds\":5,\"successThreshold\":1,\"failureThreshold\":10}",
			"resource_requests":          "{\"memory\":\"200Mi\"}",
			"stateful_set_name":          "alertmanager-main",
			"stdin":                      "0",
			"stdin_once":                 "0",
			"termination_message_path":   "/dev/termination-log",
			"termination_message_policy": "FallbackToLogsOnError",
			"tty":                        "0",
			"uid":                        "da9bb224-1bf1-5960-a83c-b77a73ea6e79",
			"volume_mounts":              "[{\"name\":\"config-volume\",\"mountPath\":\"/etc/alertmanager/config\"},{\"name\":\"alertmanager-main-db\",\"mountPath\":\"/alertmanager\"}]",
		},
		{
			"args":                       "[\"-webhook-url=http://localhost:9093/-/reload\",\"-volume-dir=/etc/alertmanager/config\"]",
			"cluster_uid":                "blah",
			"container_type":             "container",
			"creation_timestamp":         "1611191592",
			"image":                      "jimmidyson/configmap-reload:v0.3.0",
			"image_pull_policy":          "IfNotPresent",
			"labels":                     "{\"alertmanager\":\"main\"}",
			"name":                       "config-reloader",
			"namespace":                  "monitoring",
			"resource_limits":            "{\"cpu\":\"100m\",\"memory\":\"25Mi\"}",
			"stateful_set_name":          "alertmanager-main",
			"stdin":                      "0",
			"stdin_once":                 "0",
			"termination_message_path":   "/dev/termination-log",
			"termination_message_policy": "FallbackToLogsOnError",
			"tty":                        "0",
			"uid":                        "69afdc5a-a3de-59b5-8151-4103b933f2cf",
			"volume_mounts":              "[{\"name\":\"config-volume\",\"readOnly\":true,\"mountPath\":\"/etc/alertmanager/config\"}]",
		},
	}, sss)
}

func TestStatefulSetVolumesGenerate(t *testing.T) {
	sss, err := StatefulSetVolumesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"aws_elastic_block_store_partition": "0",
			"cluster_uid":                       "blah",
			"creation_timestamp":                "1611191592",
			"gce_persistent_disk_partition":     "0",
			"iscsi_discovery_chap_auth":         "0",
			"iscsi_lun":                         "0",
			"iscsi_session_chap_auth":           "0",
			"labels":                            "{\"alertmanager\":\"main\"}",
			"name":                              "config-volume",
			"namespace":                         "monitoring",
			"scale_iossl_enabled":               "0",
			"secret_default_mode":               "420",
			"secret_name":                       "alertmanager-main",
			"stateful_set_name":                 "alertmanager-main",
			"uid":                               "3c488e7e-420c-4515-b377-5dc3ee082744",
			"volume_type":                       "secret",
		},
		{
			"aws_elastic_block_store_partition": "0",
			"cluster_uid":                       "blah",
			"creation_timestamp":                "1611191592",
			"empty_dir_size_limit":              "<nil>",
			"gce_persistent_disk_partition":     "0",
			"iscsi_discovery_chap_auth":         "0",
			"iscsi_lun":                         "0",
			"iscsi_session_chap_auth":           "0",
			"labels":                            "{\"alertmanager\":\"main\"}",
			"name":                              "alertmanager-main-db",
			"namespace":                         "monitoring",
			"scale_iossl_enabled":               "0",
			"stateful_set_name":                 "alertmanager-main",
			"uid":                               "3c488e7e-420c-4515-b377-5dc3ee082744",
			"volume_type":                       "empty_dir",
		},
	}, sss)
}
