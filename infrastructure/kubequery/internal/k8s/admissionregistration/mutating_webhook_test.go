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
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMutatingWebhooksGenerate(t *testing.T) {
	i32 := int32(123)
	url := string("https://www.google.com")
	k8s.SetClient(fake.NewSimpleClientset(&v1.MutatingWebhookConfiguration{
		Webhooks: []v1.MutatingWebhook{
			{
				Name:           "mw1",
				TimeoutSeconds: &i32,
				ClientConfig:   v1.WebhookClientConfig{URL: &url},
			},
			{
				Name:           "mw2",
				TimeoutSeconds: &i32,
				ClientConfig:   v1.WebhookClientConfig{URL: &url},
			},
		},
	}), types.UID(""), "")

	mws, err := MutatingWebhooksGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"name":            "mw1",
			"timeout_seconds": "123",
			"client_config":   "{\"url\":\"https://www.google.com\"}",
		},
		{
			"name":            "mw2",
			"timeout_seconds": "123",
			"client_config":   "{\"url\":\"https://www.google.com\"}",
		},
	}, mws)
}
