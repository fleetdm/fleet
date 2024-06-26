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

type namespace struct {
	k8s.CommonFields
	v1.NamespaceStatus
}

// NamespaceColumns returns kubernetes namespace fields as Osquery table columns.
func NamespaceColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&namespace{})
}

// NamespacesGenerate generates the kubernetes namespaces as Osquery table data.
func NamespacesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		namespaces, err := k8s.GetClient().CoreV1().Namespaces().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, n := range namespaces.Items {
			item := &namespace{
				CommonFields:    k8s.GetCommonFields(n.ObjectMeta),
				NamespaceStatus: n.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if namespaces.Continue == "" {
			break
		}
		options.Continue = namespaces.Continue
	}

	return results, nil
}
