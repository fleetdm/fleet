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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
)

type info struct {
	ClusterUID  types.UID
	ClusterName string
	version.Info
}

// InfoColumns returns kubernetes info fields as Osquery table columns.
func InfoColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&info{})
}

// InfoGenerate generates the kubernetes info as Osquery table data.
func InfoGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	results := make([]map[string]string, 0)

	sv, err := k8s.GetClient().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	item := &info{
		ClusterUID:  k8s.GetClusterUID(),
		ClusterName: k8s.GetClusterName(),
		Info:        *sv,
	}
	results = append(results, k8s.ToMap(item))

	return results, nil
}
