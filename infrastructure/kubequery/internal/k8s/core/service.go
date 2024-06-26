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

type service struct {
	k8s.CommonNamespacedFields
	v1.ServiceSpec
	v1.ServiceStatus
}

// ServiceColumns returns kubernetes service fields as Osquery table columns.
func ServiceColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&service{})
}

// ServicesGenerate generates the kubernetes services as Osquery table data.
func ServicesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		services, err := k8s.GetClient().CoreV1().Services(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, s := range services.Items {
			item := &service{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(s.ObjectMeta),
				ServiceSpec:            s.Spec,
				ServiceStatus:          s.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if services.Continue == "" {
			break
		}
		options.Continue = services.Continue
	}

	return results, nil
}
