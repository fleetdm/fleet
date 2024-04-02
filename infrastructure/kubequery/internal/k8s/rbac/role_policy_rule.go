/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package rbac

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rolePolicyRule struct {
	k8s.CommonNamespacedFields
	v1.PolicyRule
}

// RolePolicyRuleColumns returns kubernetes role policy rule fields as Osquery table columns.
func RolePolicyRuleColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&rolePolicyRule{})
}

// RolePolicyRulesGenerate generates the kubernetes role policy rules as Osquery table data.
func RolePolicyRulesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		roles, err := k8s.GetClient().RbacV1().Roles(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, r := range roles.Items {
			for _, p := range r.Rules {
				item := &rolePolicyRule{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(r.ObjectMeta),
					PolicyRule:             p,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if roles.Continue == "" {
			break
		}
		options.Continue = roles.Continue
	}

	return results, nil
}
