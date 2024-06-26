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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type configmap struct {
	k8s.CommonNamespacedFields
	Immutable *bool
}

// ConfigMapColumns returns kubernetes config map fields as Osquery table columns.
func ConfigMapColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&configmap{})
}

// ConfigMapsGenerate generates the kubernetes config maps as Osquery table data.
func ConfigMapsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		configmaps, err := k8s.GetClient().CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, c := range configmaps.Items {
			item := &configmap{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(c.ObjectMeta),
				Immutable:              c.Immutable,
			}
			results = append(results, k8s.ToMap(item))
		}

		if configmaps.Continue == "" {
			break
		}
		options.Continue = configmaps.Continue
	}

	return results, nil
}
