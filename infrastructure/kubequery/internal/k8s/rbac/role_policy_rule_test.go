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
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestRolePolicyRulesGenerate(t *testing.T) {
	rprs, err := RolePolicyRulesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"Role\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-key-holder\\\",\\\"kubernetes-dashboard-certs\\\",\\\"kubernetes-dashboard-csrf\\\"],\\\"resources\\\":[\\\"secrets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\",\\\"delete\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-settings\\\"],\\\"resources\\\":[\\\"configmaps\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services\\\"],\\\"verbs\\\":[\\\"proxy\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"http:heapster:\\\",\\\"https:heapster:\\\",\\\"dashboard-metrics-scraper\\\",\\\"http:dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services/proxy\\\"],\\\"verbs\\\":[\\\"get\\\"]}]}\\n\"}",
			"api_groups":         "[\"\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"namespace":          "kube-system",
			"resource_names":     "[\"kubernetes-dashboard-key-holder\",\"kubernetes-dashboard-certs\",\"kubernetes-dashboard-csrf\"]",
			"resources":          "[\"secrets\"]",
			"uid":                "74e02baa-2c11-413f-828a-2cbe39011469",
			"verbs":              "[\"get\",\"update\",\"delete\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"Role\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-key-holder\\\",\\\"kubernetes-dashboard-certs\\\",\\\"kubernetes-dashboard-csrf\\\"],\\\"resources\\\":[\\\"secrets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\",\\\"delete\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-settings\\\"],\\\"resources\\\":[\\\"configmaps\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services\\\"],\\\"verbs\\\":[\\\"proxy\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"http:heapster:\\\",\\\"https:heapster:\\\",\\\"dashboard-metrics-scraper\\\",\\\"http:dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services/proxy\\\"],\\\"verbs\\\":[\\\"get\\\"]}]}\\n\"}",
			"api_groups":         "[\"\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"namespace":          "kube-system",
			"resource_names":     "[\"kubernetes-dashboard-settings\"]",
			"resources":          "[\"configmaps\"]",
			"uid":                "74e02baa-2c11-413f-828a-2cbe39011469",
			"verbs":              "[\"get\",\"update\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"Role\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-key-holder\\\",\\\"kubernetes-dashboard-certs\\\",\\\"kubernetes-dashboard-csrf\\\"],\\\"resources\\\":[\\\"secrets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\",\\\"delete\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-settings\\\"],\\\"resources\\\":[\\\"configmaps\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services\\\"],\\\"verbs\\\":[\\\"proxy\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"http:heapster:\\\",\\\"https:heapster:\\\",\\\"dashboard-metrics-scraper\\\",\\\"http:dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services/proxy\\\"],\\\"verbs\\\":[\\\"get\\\"]}]}\\n\"}",
			"api_groups":         "[\"\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"namespace":          "kube-system",
			"resource_names":     "[\"heapster\",\"dashboard-metrics-scraper\"]",
			"resources":          "[\"services\"]",
			"uid":                "74e02baa-2c11-413f-828a-2cbe39011469",
			"verbs":              "[\"proxy\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"Role\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\",\\\"namespace\\\":\\\"kube-system\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-key-holder\\\",\\\"kubernetes-dashboard-certs\\\",\\\"kubernetes-dashboard-csrf\\\"],\\\"resources\\\":[\\\"secrets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\",\\\"delete\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"kubernetes-dashboard-settings\\\"],\\\"resources\\\":[\\\"configmaps\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"update\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services\\\"],\\\"verbs\\\":[\\\"proxy\\\"]},{\\\"apiGroups\\\":[\\\"\\\"],\\\"resourceNames\\\":[\\\"heapster\\\",\\\"http:heapster:\\\",\\\"https:heapster:\\\",\\\"dashboard-metrics-scraper\\\",\\\"http:dashboard-metrics-scraper\\\"],\\\"resources\\\":[\\\"services/proxy\\\"],\\\"verbs\\\":[\\\"get\\\"]}]}\\n\"}",
			"api_groups":         "[\"\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"namespace":          "kube-system",
			"resource_names":     "[\"heapster\",\"http:heapster:\",\"https:heapster:\",\"dashboard-metrics-scraper\",\"http:dashboard-metrics-scraper\"]",
			"resources":          "[\"services/proxy\"]",
			"uid":                "74e02baa-2c11-413f-828a-2cbe39011469",
			"verbs":              "[\"get\"]",
		},
	}, rprs)
}
