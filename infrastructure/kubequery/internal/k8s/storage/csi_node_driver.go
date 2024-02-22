/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package storage

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type csiNodeDriver struct {
	ClusterName string
	ClusterUID  types.UID
	v1.CSINodeDriver
}

// CSINodeDriverColumns returns kubernetes CSI node driver fields as Osquery table columns.
func CSINodeDriverColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&csiNodeDriver{})
}

// CSINodeDriversGenerate generates the kubernetes CSI node drivers as Osquery table data.
func CSINodeDriversGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		nodes, err := k8s.GetClient().StorageV1().CSINodes().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, n := range nodes.Items {
			for _, d := range n.Spec.Drivers {
				item := &csiNodeDriver{
					ClusterName:   k8s.GetClusterName(),
					ClusterUID:    k8s.GetClusterUID(),
					CSINodeDriver: d,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if nodes.Continue == "" {
			break
		}
		options.Continue = nodes.Continue
	}

	return results, nil
}
