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

type ingressClass struct {
	k8s.CommonFields
	v1.IngressClassSpec
}

// IngressClassColumns returns kubernetes ingress class fields as Osquery table columns.
func IngressClassColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&ingressClass{})
}

// IngressClassesGenerate generates the kubernetes ingress classes as Osquery table data.
func IngressClassesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ics, err := k8s.GetClient().NetworkingV1().IngressClasses().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, ic := range ics.Items {
			item := &ingressClass{
				CommonFields:     k8s.GetCommonFields(ic.ObjectMeta),
				IngressClassSpec: ic.Spec,
			}
			results = append(results, k8s.ToMap(item))
		}

		if ics.Continue == "" {
			break
		}
		options.Continue = ics.Continue
	}

	return results, nil
}
