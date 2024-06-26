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
)

type csiDriver struct {
	k8s.CommonFields
	v1.CSIDriverSpec
}

// CSIDriverColumns returns kubernetes CSI driver fields as Osquery table columns.
func CSIDriverColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&csiDriver{})
}

// CSIDriversGenerate generates the kubernetes CSI drivers as Osquery table data.
func CSIDriversGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		drivers, err := k8s.GetClient().StorageV1().CSIDrivers().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, d := range drivers.Items {
			item := &csiDriver{
				CommonFields:  k8s.GetCommonFields(d.ObjectMeta),
				CSIDriverSpec: d.Spec,
			}
			results = append(results, k8s.ToMap(item))
		}

		if drivers.Continue == "" {
			break
		}
		options.Continue = drivers.Continue
	}

	return results, nil
}
