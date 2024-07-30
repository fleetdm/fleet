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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type csiStorageCapacity struct {
	k8s.CommonNamespacedFields
	NodeTopology     *metav1.LabelSelector
	StorageClassName string
	Capacity         *resource.Quantity
}

// CSIStorageCapacityColumns returns kubernetes CSI storage capacity fields as Osquery table columns.
func CSIStorageCapacityColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&csiStorageCapacity{})
}

// CSIStorageCapacitiesGenerate generates the kubernetes CSI storage capacities as Osquery table data.
func CSIStorageCapacitiesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		scs, err := k8s.GetClient().StorageV1alpha1().CSIStorageCapacities(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, sc := range scs.Items {
			item := &csiStorageCapacity{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(sc.ObjectMeta),
				NodeTopology:           sc.NodeTopology,
				StorageClassName:       sc.StorageClassName,
				Capacity:               sc.Capacity,
			}
			results = append(results, k8s.ToMap(item))
		}

		if scs.Continue == "" {
			break
		}
		options.Continue = scs.Continue
	}

	return results, nil
}
