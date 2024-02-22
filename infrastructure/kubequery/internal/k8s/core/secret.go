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

type secret struct {
	k8s.CommonNamespacedFields
	Immutable *bool
	Type      v1.SecretType
}

// SecretColumns returns kubernetes secret fields as Osquery table columns.
func SecretColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&secret{})
}

// SecretsGenerate generates the kubernetes secrets as Osquery table data.
func SecretsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		secrets, err := k8s.GetClient().CoreV1().Secrets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, s := range secrets.Items {
			item := &secret{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(s.ObjectMeta),
				Immutable:              s.Immutable,
				Type:                   s.Type,
			}
			results = append(results, k8s.ToMap(item))
		}

		if secrets.Continue == "" {
			break
		}
		options.Continue = secrets.Continue
	}

	return results, nil
}
