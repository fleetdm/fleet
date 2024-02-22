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

type serviceAccount struct {
	k8s.CommonNamespacedFields
	Secrets                      []v1.ObjectReference
	ImagePullSecrets             []v1.LocalObjectReference
	AutomountServiceAccountToken *bool
}

// ServiceAccountColumns returns kubernetes service account fields as Osquery table columns.
func ServiceAccountColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&serviceAccount{})
}

// ServiceAccountsGenerate generates the kubernetes service accounts as Osquery table data.
func ServiceAccountsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		sas, err := k8s.GetClient().CoreV1().ServiceAccounts(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, sa := range sas.Items {
			item := &serviceAccount{
				CommonNamespacedFields:       k8s.GetCommonNamespacedFields(sa.ObjectMeta),
				Secrets:                      sa.Secrets,
				ImagePullSecrets:             sa.ImagePullSecrets,
				AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
			}
			results = append(results, k8s.ToMap(item))
		}

		if sas.Continue == "" {
			break
		}
		options.Continue = sas.Continue
	}

	return results, nil
}
