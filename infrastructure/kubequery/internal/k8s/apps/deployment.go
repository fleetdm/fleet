/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package apps

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deployment struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.DeploymentStatus
	DeploymentReplicas      *int32
	Selector                *metav1.LabelSelector
	Strategy                v1.DeploymentStrategy
	MinReadySeconds         int32
	RevisionHistoryLimit    *int32
	Paused                  bool
	ProgressDeadlineSeconds *int32
}

// DeploymentColumns returns kubernetes deployment fields as Osquery table columns.
func DeploymentColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&deployment{})
}

// DeploymentsGenerate generates the kubernetes deployments as Osquery table data.
func DeploymentsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ds, err := k8s.GetClient().AppsV1().Deployments(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, d := range ds.Items {
			item := &deployment{
				CommonNamespacedFields:  k8s.GetCommonNamespacedFields(d.ObjectMeta),
				CommonPodFields:         k8s.GetCommonPodFields(d.Spec.Template.Spec),
				DeploymentReplicas:      d.Spec.Replicas,
				Selector:                d.Spec.Selector,
				Strategy:                d.Spec.Strategy,
				MinReadySeconds:         d.Spec.MinReadySeconds,
				RevisionHistoryLimit:    d.Spec.RevisionHistoryLimit,
				Paused:                  d.Spec.Paused,
				ProgressDeadlineSeconds: d.Spec.ProgressDeadlineSeconds,
				DeploymentStatus:        d.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if ds.Continue == "" {
			break
		}
		options.Continue = ds.Continue
	}

	return results, nil
}

type deploymentContainer struct {
	k8s.CommonNamespacedFields
	k8s.CommonContainerFields
	DeploymentName string
	ContainerType  string
}

// DeploymentContainerColumns returns kubernetes deployment container fields as Osquery table columns.
func DeploymentContainerColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&deploymentContainer{})
}

// DeploymentContainersGenerate generates the kubernetes deployment containers as Osquery table data.
func DeploymentContainersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ds, err := k8s.GetClient().AppsV1().Deployments(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, d := range ds.Items {
			for _, c := range d.Spec.Template.Spec.InitContainers {
				item := &deploymentContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(d.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					DeploymentName:         d.Name,
					ContainerType:          "init",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range d.Spec.Template.Spec.Containers {
				item := &deploymentContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(d.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					DeploymentName:         d.Name,
					ContainerType:          "container",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range d.Spec.Template.Spec.EphemeralContainers {
				item := &deploymentContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(d.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonEphemeralContainerFields(c),
					DeploymentName:         d.Name,
					ContainerType:          "ephemeral",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if ds.Continue == "" {
			break
		}
		options.Continue = ds.Continue
	}

	return results, nil
}

type deploymentVolume struct {
	k8s.CommonNamespacedFields
	k8s.CommonVolumeFields
	DeploymentName string
}

// DeploymentVolumeColumns returns kubernetes deployment volume fields as Osquery table columns.
func DeploymentVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&deploymentVolume{})
}

// DeploymentVolumesGenerate generates the kubernetes deployment volumes as Osquery table data.
func DeploymentVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		ds, err := k8s.GetClient().AppsV1().Deployments(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, d := range ds.Items {
			for _, v := range d.Spec.Template.Spec.Volumes {
				item := &deploymentVolume{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(d.ObjectMeta),
					CommonVolumeFields:     k8s.GetCommonVolumeFields(v),
					DeploymentName:         d.Name,
				}
				item.Name = v.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if ds.Continue == "" {
			break
		}
		options.Continue = ds.Continue
	}

	return results, nil
}
