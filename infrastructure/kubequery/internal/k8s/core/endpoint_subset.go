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

type endpointSubset struct {
	k8s.CommonNamespacedFields
	v1.EndpointSubset
}

// EndpointSubsetColumns returns kubernetes endpoint subset fields as Osquery table columns.
func EndpointSubsetColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&endpointSubset{})
}

// EndpointSubsetsGenerate generates the kubernetes endpoint subsets as Osquery table data.
func EndpointSubsetsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		endpoints, err := k8s.GetClient().CoreV1().Endpoints(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, e := range endpoints.Items {
			for _, s := range e.Subsets {
				item := &endpointSubset{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(e.ObjectMeta),
					EndpointSubset:         s,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if endpoints.Continue == "" {
			break
		}
		options.Continue = endpoints.Continue
	}

	return results, nil
}
