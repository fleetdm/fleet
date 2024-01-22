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

type cronJob struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.CronJobStatus
	Schedule                   string
	StartingDeadlineSeconds    *int64
	ConcurrencyPolicy          v1.ConcurrencyPolicy
	Suspend                    *bool
	SuccessfulJobsHistoryLimit *int32
	FailedJobsHistoryLimit     *int32
	Parallelism                *int32
	Completions                *int32
	JobActiveDeadlineSeconds   *int64
	BackoffLimit               *int32
	Selector                   *metav1.LabelSelector
	ManualSelector             *bool
	TTLSecondsAfterFinished    *int32
}

// CronJobColumns returns kubernetes cron job fields as Osquery table columns.
func CronJobColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&cronJob{})
}

// CronJobsGenerate generates the kubernetes cron jobs as Osquery table data.
func CronJobsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		cjs, err := k8s.GetClient().BatchV1().CronJobs(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, cj := range cjs.Items {
			item := &cronJob{
				CommonNamespacedFields:     k8s.GetCommonNamespacedFields(cj.ObjectMeta),
				CommonPodFields:            k8s.GetCommonPodFields(cj.Spec.JobTemplate.Spec.Template.Spec),
				CronJobStatus:              cj.Status,
				Schedule:                   cj.Spec.Schedule,
				StartingDeadlineSeconds:    cj.Spec.StartingDeadlineSeconds,
				ConcurrencyPolicy:          cj.Spec.ConcurrencyPolicy,
				Suspend:                    cj.Spec.Suspend,
				SuccessfulJobsHistoryLimit: cj.Spec.SuccessfulJobsHistoryLimit,
				FailedJobsHistoryLimit:     cj.Spec.FailedJobsHistoryLimit,
				Parallelism:                cj.Spec.JobTemplate.Spec.Parallelism,
				Completions:                cj.Spec.JobTemplate.Spec.Completions,
				JobActiveDeadlineSeconds:   cj.Spec.JobTemplate.Spec.ActiveDeadlineSeconds,
				BackoffLimit:               cj.Spec.JobTemplate.Spec.BackoffLimit,
				Selector:                   cj.Spec.JobTemplate.Spec.Selector,
				ManualSelector:             cj.Spec.JobTemplate.Spec.ManualSelector,
				TTLSecondsAfterFinished:    cj.Spec.JobTemplate.Spec.TTLSecondsAfterFinished,
			}
			results = append(results, k8s.ToMap(item))
		}

		if cjs.Continue == "" {
			break
		}
		options.Continue = cjs.Continue
	}

	return results, nil
}
