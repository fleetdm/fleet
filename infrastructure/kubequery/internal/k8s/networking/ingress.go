/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package networking

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ingress struct {
	k8s.CommonNamespacedFields
	v1.IngressSpec
	v1.IngressStatus
}

// IngressColumns returns kubernetes ingress fields as Osquery table columns.
func IngressColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&ingress{})
}

// IngressesGenerate generates the kubernetes ingresses as Osquery table data.
func IngressesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ingresses, err := k8s.GetClient().NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, i := range ingresses.Items {
			item := &ingress{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(i.ObjectMeta),
				IngressSpec:            i.Spec,
				IngressStatus:          i.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if ingresses.Continue == "" {
			break
		}
		options.Continue = ingresses.Continue
	}

	return results, nil
}
