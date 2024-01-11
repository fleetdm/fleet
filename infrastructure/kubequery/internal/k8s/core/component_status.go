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
	"k8s.io/apimachinery/pkg/types"
)

type componentStatus struct {
	ClusterName string
	ClusterUID  types.UID
	Name        string
	v1.ComponentCondition
}

// ComponentStatusColumns returns kubernetes component status fields as Osquery table columns.
func ComponentStatusColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&componentStatus{})
}

// ComponentStatusesGenerate generates the kubernetes config maps as Osquery table data.
func ComponentStatusesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		css, err := k8s.GetClient().CoreV1().ComponentStatuses().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, cs := range css.Items {
			for _, cc := range cs.Conditions {
				item := &componentStatus{
					ClusterName:        k8s.GetClusterName(),
					ClusterUID:         k8s.GetClusterUID(),
					Name:               cs.ObjectMeta.Name,
					ComponentCondition: cc,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if css.Continue == "" {
			break
		}
		options.Continue = css.Continue
	}

	return results, nil
}
