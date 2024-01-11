/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package autoscaling

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHorizontalPodAutoscalerGenerate(t *testing.T) {
	i32 := int32(456)
	i64 := int64(123)
	k8s.SetClient(fake.NewSimpleClientset(&v1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa1",
			Namespace: "n123",
			UID:       types.UID("1234"),
			Labels:    map[string]string{"a": "b"},
		},
		Spec: v1.HorizontalPodAutoscalerSpec{
			MinReplicas:                    &i32,
			MaxReplicas:                    i32,
			TargetCPUUtilizationPercentage: &i32,
			ScaleTargetRef: v1.CrossVersionObjectReference{
				Name: "blah",
			},
		},
		Status: v1.HorizontalPodAutoscalerStatus{
			ObservedGeneration:              &i64,
			LastScaleTime:                   &metav1.Time{},
			CurrentReplicas:                 i32,
			DesiredReplicas:                 i32,
			CurrentCPUUtilizationPercentage: &i32,
		},
	}), types.UID("hello"), "")

	hpas, err := HorizontalPodAutoscalerGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"cluster_uid":                        "hello",
			"creation_timestamp":                 "0",
			"current_cpu_utilization_percentage": "456",
			"current_replicas":                   "456",
			"desired_replicas":                   "456",
			"labels":                             "{\"a\":\"b\"}",
			"last_scale_time":                    "0",
			"max_replicas":                       "456",
			"min_replicas":                       "456",
			"name":                               "hpa1",
			"namespace":                          "n123",
			"observed_generation":                "123",
			"scale_target_ref":                   "{\"kind\":\"\",\"name\":\"blah\"}",
			"target_cpu_utilization_percentage":  "456",
			"uid":                                "1234",
		},
	}, hpas)
}
