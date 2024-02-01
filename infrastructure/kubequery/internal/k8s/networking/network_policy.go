/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package networking

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type networkPolicy struct {
	k8s.CommonNamespacedFields
	PodSelector metav1.LabelSelector
	PolicyTypes []v1.PolicyType
	Type        string
	Ports       []v1.NetworkPolicyPort
	FromTo      []v1.NetworkPolicyPeer
}

// NetworkPolicyColumns returns kubernetes network policy fields as Osquery table columns.
func NetworkPolicyColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&networkPolicy{})
}

// NetworkPoliciesGenerate generates the kubernetes network policies as Osquery table data.
func NetworkPoliciesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		nps, err := k8s.GetClient().NetworkingV1().NetworkPolicies(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, np := range nps.Items {
			for _, i := range np.Spec.Ingress {
				item := &networkPolicy{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(np.ObjectMeta),
					PodSelector:            np.Spec.PodSelector,
					PolicyTypes:            np.Spec.PolicyTypes,
					Type:                   "ingress",
					Ports:                  i.Ports,
					FromTo:                 i.From,
				}
				results = append(results, k8s.ToMap(item))
			}
			for _, e := range np.Spec.Egress {
				item := &networkPolicy{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(np.ObjectMeta),
					PodSelector:            np.Spec.PodSelector,
					PolicyTypes:            np.Spec.PolicyTypes,
					Type:                   "egress",
					Ports:                  e.Ports,
					FromTo:                 e.To,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if nps.Continue == "" {
			break
		}
		options.Continue = nps.Continue
	}

	return results, nil
}
