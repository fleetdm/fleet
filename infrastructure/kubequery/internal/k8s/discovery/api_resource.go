/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package discovery

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type apiResource struct {
	ClusterName string
	ClusterUID  types.UID
	metav1.APIResource
	GroupVersion string
}

// APIResourceColumns returns kubernetes API resource fields as Osquery table columns.
func APIResourceColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&apiResource{})
}

// APIResourcesGenerate generates the kubernetes API resources as Osquery table data.
func APIResourcesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	results := make([]map[string]string, 0)

	sr, err := k8s.GetClient().Discovery().ServerResources()
	if err != nil {
		return nil, err
	}

	for _, rl := range sr {
		for _, r := range rl.APIResources {
			item := &apiResource{
				ClusterName:  k8s.GetClusterName(),
				ClusterUID:   k8s.GetClusterUID(),
				GroupVersion: rl.GroupVersion,
				APIResource:  r,
			}
			results = append(results, k8s.ToMap(item))
		}
	}

	return results, nil
}
