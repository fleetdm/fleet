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

type limitRange struct {
	k8s.CommonNamespacedFields
	v1.LimitRangeItem
}

// LimitRangeColumns returns kubernetes limit range fields as Osquery table columns.
func LimitRangeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&limitRange{})
}

// LimitRangesGenerate generates the kubernetes limit ranges as Osquery table data.
func LimitRangesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ranges, err := k8s.GetClient().CoreV1().LimitRanges(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, r := range ranges.Items {
			for _, i := range r.Spec.Limits {
				item := &limitRange{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(r.ObjectMeta),
					LimitRangeItem:         i,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if ranges.Continue == "" {
			break
		}
		options.Continue = ranges.Continue
	}

	return results, nil
}
