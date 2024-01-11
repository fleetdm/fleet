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

func TestCronJobsGenerate(t *testing.T) {
	i32 := int32(456)
	i64 := int64(123)
	b := bool(true)
	k8s.SetClient(fake.NewSimpleClientset(&v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cj1",
			Namespace: "n123",
			UID:       types.UID("1234"),
			Labels:    map[string]string{"a": "b"},
		},
		Spec: v1.CronJobSpec{
			Schedule:                   "s1",
			StartingDeadlineSeconds:    &i64,
			ConcurrencyPolicy:          v1.AllowConcurrent,
			Suspend:                    &b,
			SuccessfulJobsHistoryLimit: &i32,
			FailedJobsHistoryLimit:     &i32,
			JobTemplate: v1.JobTemplateSpec{
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
			},
		},
		Status: v1.CronJobStatus{
			LastScheduleTime: &metav1.Time{},
		},
	}), types.UID("hello"), "")

	cjs, err := CronJobsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"backoff_limit":                 "456",
			"cluster_uid":                   "hello",
			"completions":                   "456",
			"concurrency_policy":            "Allow",
			"creation_timestamp":            "0",
			"failed_jobs_history_limit":     "456",
			"host_ipc":                      "0",
			"host_network":                  "0",
			"host_pid":                      "0",
			"job_active_deadline_seconds":   "123",
			"labels":                        "{\"a\":\"b\"}",
			"last_schedule_time":            "0",
			"manual_selector":               "1",
			"name":                          "cj1",
			"namespace":                     "n123",
			"parallelism":                   "456",
			"schedule":                      "s1",
			"starting_deadline_seconds":     "123",
			"successful_jobs_history_limit": "456",
			"suspend":                       "1",
			"ttl_seconds_after_finished":    "456",
			"uid":                           "1234",
		},
	}, cjs)
}
