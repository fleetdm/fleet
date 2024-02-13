/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package batch

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestJobsGenerate(t *testing.T) {
	i32 := int32(456)
	i64 := int64(123)
	b := bool(true)
	k8s.SetClient(fake.NewSimpleClientset(&v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job1",
			Namespace: "n123",
			UID:       types.UID("1234"),
			Labels:    map[string]string{"a": "b"},
		},
		Spec: v1.JobSpec{
			Parallelism:             &i32,
			Completions:             &i32,
			ActiveDeadlineSeconds:   &i64,
			BackoffLimit:            &i32,
			ManualSelector:          &b,
			TTLSecondsAfterFinished: &i32,
		},
		Status: v1.JobStatus{
			StartTime:      &metav1.Time{},
			Active:         i32,
			Succeeded:      i32,
			Failed:         i32,
			CompletionTime: nil,
			Conditions: []v1.JobCondition{
				{
					Type:               v1.JobComplete,
					Status:             metav1.DryRunAll,
					LastTransitionTime: metav1.Time{},
					Reason:             "reason",
					Message:            "message",
				},
			},
		},
	}), types.UID("hello"), "")

	js, err := JobsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"active":                      "456",
			"backoff_limit":               "456",
			"cluster_uid":                 "hello",
			"completions":                 "456",
			"conditions":                  "[{\"type\":\"Complete\",\"status\":\"All\",\"lastProbeTime\":null,\"lastTransitionTime\":null,\"reason\":\"reason\",\"message\":\"message\"}]",
			"creation_timestamp":          "0",
			"failed":                      "456",
			"host_ipc":                    "0",
			"host_network":                "0",
			"host_pid":                    "0",
			"job_active_deadline_seconds": "123",
			"labels":                      "{\"a\":\"b\"}",
			"manual_selector":             "1",
			"name":                        "job1",
			"namespace":                   "n123",
			"parallelism":                 "456",
			"start_time":                  "0",
			"succeeded":                   "456",
			"ttl_seconds_after_finished":  "456",
			"uid":                         "1234",
		},
	}, js)
}
