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

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type job struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.JobStatus
	Parallelism              *int32
	Completions              *int32
	JobActiveDeadlineSeconds *int64
	BackoffLimit             *int32
	Selector                 *metav1.LabelSelector
	ManualSelector           *bool
	TTLSecondsAfterFinished  *int32
}

// JobColumns returns kubernetes job fields as Osquery table columns.
func JobColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&job{})
}

// JobsGenerate generates the kubernetes jobs as Osquery table data.
func JobsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		jobs, err := k8s.GetClient().BatchV1().Jobs(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, j := range jobs.Items {
			item := &job{
				CommonNamespacedFields:   k8s.GetCommonNamespacedFields(j.ObjectMeta),
				CommonPodFields:          k8s.GetCommonPodFields(j.Spec.Template.Spec),
				JobStatus:                j.Status,
				Parallelism:              j.Spec.Parallelism,
				Completions:              j.Spec.Completions,
				JobActiveDeadlineSeconds: j.Spec.ActiveDeadlineSeconds,
				BackoffLimit:             j.Spec.BackoffLimit,
				Selector:                 j.Spec.Selector,
				ManualSelector:           j.Spec.ManualSelector,
				TTLSecondsAfterFinished:  j.Spec.TTLSecondsAfterFinished,
			}
			results = append(results, k8s.ToMap(item))
		}

		if jobs.Continue == "" {
			break
		}
		options.Continue = jobs.Continue
	}

	return results, nil
}
