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

type node struct {
	k8s.CommonFields
	v1.NodeSpec
	v1.NodeStatus
}

// NodeColumns returns kubernetes node fields as Osquery table columns.
func NodeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&node{})
}

// NodesGenerate generates the kubernetes nodes as Osquery table data.
func NodesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		nodes, err := k8s.GetClient().CoreV1().Nodes().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, n := range nodes.Items {
			item := &node{
				CommonFields: k8s.GetCommonFields(n.ObjectMeta),
				NodeSpec:     n.Spec,
				NodeStatus:   n.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if nodes.Continue == "" {
			break
		}
		options.Continue = nodes.Continue
	}

	return results, nil
}
