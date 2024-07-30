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

func TestClusterRolePolicyRulesGenerate(t *testing.T) {
	crprs, err := ClusterRolePolicyRulesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"configmaps\",\"endpoints\",\"namespaces\",\"nodes\",\"pods\",\"pods/log\",\"replicationcontrollers\",\"services\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"extensions\",\"apps\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"deployments\",\"replicasets\",\"statefulsets\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"autoscaling\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"horizontalpodautoscalers\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"batch\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"cronjobs\",\"jobs\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"config.istio.io\",\"networking.istio.io\",\"authentication.istio.io\",\"rbac.istio.io\",\"security.istio.io\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"*\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"app\\\":\\\"kiali\\\",\\\"chart\\\":\\\"kiali\\\",\\\"heritage\\\":\\\"Tiller\\\",\\\"release\\\":\\\"istio\\\"},\\\"name\\\":\\\"kiali-viewer\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"\\\"],\\\"resources\\\":[\\\"configmaps\\\",\\\"endpoints\\\",\\\"namespaces\\\",\\\"nodes\\\",\\\"pods\\\",\\\"pods/log\\\",\\\"replicationcontrollers\\\",\\\"services\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"extensions\\\",\\\"apps\\\"],\\\"resources\\\":[\\\"deployments\\\",\\\"replicasets\\\",\\\"statefulsets\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"autoscaling\\\"],\\\"resources\\\":[\\\"horizontalpodautoscalers\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"batch\\\"],\\\"resources\\\":[\\\"cronjobs\\\",\\\"jobs\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"config.istio.io\\\",\\\"networking.istio.io\\\",\\\"authentication.istio.io\\\",\\\"rbac.istio.io\\\",\\\"security.istio.io\\\"],\\\"resources\\\":[\\\"*\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]},{\\\"apiGroups\\\":[\\\"monitoring.kiali.io\\\"],\\\"resources\\\":[\\\"monitoringdashboards\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\"]}]}\\n\"}",
			"api_groups":         "[\"monitoring.kiali.io\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611191143",
			"labels":             "{\"app\":\"kiali\",\"chart\":\"kiali\",\"heritage\":\"Tiller\",\"release\":\"istio\"}",
			"name":               "kiali-viewer",
			"resources":          "[\"monitoringdashboards\"]",
			"uid":                "b5d5ca79-f4e3-4478-b954-fe62a146f279",
			"verbs":              "[\"get\",\"list\"]",
		},
		{
			"annotations":        "{\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"rbac.authorization.k8s.io/v1\\\",\\\"kind\\\":\\\"ClusterRole\\\",\\\"metadata\\\":{\\\"annotations\\\":{},\\\"labels\\\":{\\\"k8s-app\\\":\\\"kubernetes-dashboard\\\"},\\\"name\\\":\\\"kubernetes-dashboard\\\"},\\\"rules\\\":[{\\\"apiGroups\\\":[\\\"metrics.k8s.io\\\"],\\\"resources\\\":[\\\"pods\\\",\\\"nodes\\\"],\\\"verbs\\\":[\\\"get\\\",\\\"list\\\",\\\"watch\\\"]}]}\\n\"}",
			"api_groups":         "[\"metrics.k8s.io\"]",
			"cluster_uid":        "a7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp": "1611190911",
			"labels":             "{\"k8s-app\":\"kubernetes-dashboard\"}",
			"name":               "kubernetes-dashboard",
			"resources":          "[\"pods\",\"nodes\"]",
			"uid":                "5afb084d-e4da-4207-844d-d3a2e002ecda",
			"verbs":              "[\"get\",\"list\",\"watch\"]",
		},
	}, crprs)
}
