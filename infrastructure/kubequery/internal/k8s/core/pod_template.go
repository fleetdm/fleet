/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package core

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podTemplate struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
}

// PodTemplateColumns returns kubernetes pod template fields as Osquery table columns.
func PodTemplateColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podTemplate{})
}

// PodTemplatesGenerate generates the kubernetes pod templates as Osquery table data.
func PodTemplatesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pts, err := k8s.GetClient().CoreV1().PodTemplates(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, pt := range pts.Items {
			item := &podTemplate{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(pt.ObjectMeta),
				CommonPodFields:        k8s.GetCommonPodFields(pt.Template.Spec),
			}
			results = append(results, k8s.ToMap(item))
		}

		if pts.Continue == "" {
			break
		}
		options.Continue = pts.Continue
	}

	return results, nil
}

type podTemplateContainer struct {
	k8s.CommonNamespacedFields
	k8s.CommonContainerFields
	PodTemplateName string
	ContainerType   string
}

// PodTemplateContainerColumns returns kubernetes pod template container fields as Osquery table columns.
func PodTemplateContainerColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podTemplateContainer{})
}

func createPodTemplateContainer(pt v1.PodTemplate, c v1.Container, containerType string) *podTemplateContainer {
	item := &podTemplateContainer{
		CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(pt.ObjectMeta, c.Name),
		CommonContainerFields:  k8s.GetCommonContainerFields(c),
		PodTemplateName:        pt.Name,
		ContainerType:          containerType,
	}
	item.Name = c.Name
	return item
}

func createPodTemplateEphemeralContainer(pt v1.PodTemplate, c v1.EphemeralContainer) *podTemplateContainer {
	item := &podTemplateContainer{
		CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(pt.ObjectMeta, c.Name),
		CommonContainerFields:  k8s.GetCommonEphemeralContainerFields(c),
		PodTemplateName:        pt.Name,
		ContainerType:          "ephemeral",
	}
	item.Name = c.Name
	return item
}

// PodTemplateContainersGenerate generates the kubernetes pod template containers as Osquery table data.
func PodTemplateContainersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pts, err := k8s.GetClient().CoreV1().PodTemplates(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, pt := range pts.Items {
			for _, c := range pt.Template.Spec.InitContainers {
				item := createPodTemplateContainer(pt, c, "init")
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range pt.Template.Spec.Containers {
				item := createPodTemplateContainer(pt, c, "container")
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range pt.Template.Spec.EphemeralContainers {
				item := createPodTemplateEphemeralContainer(pt, c)
				results = append(results, k8s.ToMap(item))
			}
		}

		if pts.Continue == "" {
			break
		}
		options.Continue = pts.Continue
	}

	return results, nil
}

type podTemplateVolume struct {
	k8s.CommonNamespacedFields
	k8s.CommonVolumeFields
	PodTemplateName string
}

// PodTemplateVolumeColumns returns kubernetes pod template volume fields as Osquery table columns.
func PodTemplateVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podTemplateVolume{})
}

// PodTemplateVolumesGenerate generates the kubernetes pod template volumes as Osquery table data.
func PodTemplateVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pts, err := k8s.GetClient().CoreV1().PodTemplates(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, pt := range pts.Items {
			for _, v := range pt.Template.Spec.Volumes {
				item := &podTemplateVolume{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(pt.ObjectMeta),
					CommonVolumeFields:     k8s.GetCommonVolumeFields(v),
					PodTemplateName:        pt.Name,
				}
				item.Name = v.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if pts.Continue == "" {
			break
		}
		options.Continue = pts.Continue
	}

	return results, nil
}
