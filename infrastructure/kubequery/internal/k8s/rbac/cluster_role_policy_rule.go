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

type clusterRolePolicyRule struct {
	k8s.CommonFields
	v1.PolicyRule
	AggregationRule *v1.AggregationRule
}

// ClusterRolePolicyRuleColumns returns kubernetes cluster role policy rule fields as Osquery table columns.
func ClusterRolePolicyRuleColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&clusterRolePolicyRule{})
}

// ClusterRolePolicyRulesGenerate generates the kubernetes cluster role policy rules as Osquery table data.
func ClusterRolePolicyRulesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		crs, err := k8s.GetClient().RbacV1().ClusterRoles().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, cr := range crs.Items {
			for _, r := range cr.Rules {
				item := &clusterRolePolicyRule{
					CommonFields:    k8s.GetCommonFields(cr.ObjectMeta),
					PolicyRule:      r,
					AggregationRule: cr.AggregationRule,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if crs.Continue == "" {
			break
		}
		options.Continue = crs.Continue
	}

	return results, nil
}
