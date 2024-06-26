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

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resourceQuota struct {
	k8s.CommonNamespacedFields
	v1.ResourceQuotaSpec
	StatusHard v1.ResourceList
	StatusUsed v1.ResourceList
}

// ResourceQuotaColumns returns kubernetes resource quota fields as Osquery table columns.
func ResourceQuotaColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&resourceQuota{})
}

// ResourceQuotasGenerate generates the kubernetes resource quotas as Osquery table data.
func ResourceQuotasGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		quotas, err := k8s.GetClient().CoreV1().ResourceQuotas(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, q := range quotas.Items {
			item := &resourceQuota{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(q.ObjectMeta),
				ResourceQuotaSpec:      q.Spec,
				StatusHard:             q.Status.Hard,
				StatusUsed:             q.Status.Used,
			}
			results = append(results, k8s.ToMap(item))
		}

		if quotas.Continue == "" {
			break
		}
		options.Continue = quotas.Continue
	}

	return results, nil
}
