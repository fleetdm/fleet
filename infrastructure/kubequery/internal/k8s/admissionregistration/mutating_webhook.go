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

type mutatingWebhook struct {
	ClusterName string
	ClusterUID  types.UID
	v1.MutatingWebhook
}

// MutatingWebhookColumns returns kubernetes mutating webhook fields as Osquery table columns.
func MutatingWebhookColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&mutatingWebhook{})
}

// MutatingWebhooksGenerate generates the mutating webhook Osquery table data.
func MutatingWebhooksGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		mwcs, err := k8s.GetClient().AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, mwc := range mwcs.Items {
			for _, mw := range mwc.Webhooks {
				item := &mutatingWebhook{
					ClusterName:     k8s.GetClusterName(),
					ClusterUID:      k8s.GetClusterUID(),
					MutatingWebhook: mw,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if mwcs.Continue == "" {
			break
		}
		options.Continue = mwcs.Continue
	}

	return results, nil
}
