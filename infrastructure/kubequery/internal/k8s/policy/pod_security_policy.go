/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package policy

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podSecurityPolicy struct {
	k8s.CommonFields
	v1beta1.PodSecurityPolicySpec
}

// PodSecurityPolicyColumns returns kubernetes pod security policy fields as Osquery table columns.
func PodSecurityPolicyColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podSecurityPolicy{})
}

// PodSecurityPoliciesGenerate generates the kubernetes pod security policies as Osquery table data.
func PodSecurityPoliciesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		psps, err := k8s.GetClient().PolicyV1beta1().PodSecurityPolicies().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, psp := range psps.Items {
			item := &podSecurityPolicy{
				CommonFields:          k8s.GetCommonFields(psp.ObjectMeta),
				PodSecurityPolicySpec: psp.Spec,
			}
			results = append(results, k8s.ToMap(item))
		}

		if psps.Continue == "" {
			break
		}
		options.Continue = psps.Continue
	}

	return results, nil
}
