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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type storageClass struct {
	k8s.CommonFields
	Provisioner          string
	Parameters           map[string]string
	ReclaimPolicy        *corev1.PersistentVolumeReclaimPolicy
	MountOptions         []string
	AllowVolumeExpansion *bool
	VolumeBindingMode    *v1.VolumeBindingMode
	AllowedTopologies    []corev1.TopologySelectorTerm
}

// SCClassColumns returns kubernetes storage class fields as Osquery table columns.
func SCClassColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&storageClass{})
}

// SCClassesGenerate generates the kubernetes storage classes as Osquery table data.
func SCClassesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		classes, err := k8s.GetClient().StorageV1().StorageClasses().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, c := range classes.Items {
			item := &storageClass{
				CommonFields:         k8s.GetCommonFields(c.ObjectMeta),
				Provisioner:          c.Provisioner,
				Parameters:           c.Parameters,
				ReclaimPolicy:        c.ReclaimPolicy,
				MountOptions:         c.MountOptions,
				AllowVolumeExpansion: c.AllowVolumeExpansion,
				VolumeBindingMode:    c.VolumeBindingMode,
				AllowedTopologies:    c.AllowedTopologies,
			}
			results = append(results, k8s.ToMap(item))
		}

		if classes.Continue == "" {
			break
		}
		options.Continue = classes.Continue
	}

	return results, nil
}
