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

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type horizontalPodAutoscaler struct {
	k8s.CommonNamespacedFields
	v1.HorizontalPodAutoscalerSpec
	v1.HorizontalPodAutoscalerStatus
}

// HorizontalPodAutoscalersColumns returns kubernetes horizontal pod autoscaler fields as Osquery table columns.
func HorizontalPodAutoscalersColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&horizontalPodAutoscaler{})
}

// HorizontalPodAutoscalerGenerate generates the kubernetes horizontal pod autoscalers as Osquery table data.
func HorizontalPodAutoscalerGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		hpas, err := k8s.GetClient().AutoscalingV1().HorizontalPodAutoscalers(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, hpa := range hpas.Items {
			item := &horizontalPodAutoscaler{
				CommonNamespacedFields:        k8s.GetCommonNamespacedFields(hpa.ObjectMeta),
				HorizontalPodAutoscalerSpec:   hpa.Spec,
				HorizontalPodAutoscalerStatus: hpa.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if hpas.Continue == "" {
			break
		}
		options.Continue = hpas.Continue
	}

	return results, nil
}
