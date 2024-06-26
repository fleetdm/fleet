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

type persistentVolumeClaim struct {
	k8s.CommonFields
	v1.PersistentVolumeClaimSpec
	Phase      v1.PersistentVolumeClaimPhase
	Capacity   v1.ResourceList
	Conditions []v1.PersistentVolumeClaimCondition
}

// PersistentVolumeClaimColumns returns kubernetes persistent volume claim fields as Osquery table columns.
func PersistentVolumeClaimColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&persistentVolumeClaim{})
}

// PersistentVolumeClaimsGenerate generates the kubernetes persistent volume claims as Osquery table data.
func PersistentVolumeClaimsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pvcs, err := k8s.GetClient().CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, pvc := range pvcs.Items {
			item := &persistentVolumeClaim{
				CommonFields:              k8s.GetCommonFields(pvc.ObjectMeta),
				PersistentVolumeClaimSpec: pvc.Spec,
				Phase:                     pvc.Status.Phase,
				Capacity:                  pvc.Status.Capacity,
				Conditions:                pvc.Status.Conditions,
			}
			results = append(results, k8s.ToMap(item))
		}

		if pvcs.Continue == "" {
			break
		}
		options.Continue = pvcs.Continue
	}

	return results, nil
}
