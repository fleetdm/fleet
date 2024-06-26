/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package admissionregistration

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type validatingWebhook struct {
	ClusterName string
	ClusterUID  types.UID
	v1.ValidatingWebhook
}

// ValidatingWebhookColumns returns kubernetes validating webhook fields as Osquery table columns.
func ValidatingWebhookColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&validatingWebhook{})
}

// ValidatingWebhooksGenerate generates the kubernetes validating webhooks as Osquery table data.
func ValidatingWebhooksGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		vwcs, err := k8s.GetClient().AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, vwc := range vwcs.Items {
			for _, vw := range vwc.Webhooks {
				item := &validatingWebhook{
					ClusterName:       k8s.GetClusterName(),
					ClusterUID:        k8s.GetClusterUID(),
					ValidatingWebhook: vw,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if vwcs.Continue == "" {
			break
		}
		options.Continue = vwcs.Continue
	}

	return results, nil
}
