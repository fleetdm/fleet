/**
 * Copyright (c) 2020-present, The kubequery authoss
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type statefulSet struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.StatefulSetStatus
	StatefulSetReplicas  *int32
	Selector             *metav1.LabelSelector
	VolumeClaimTemplates []corev1.PersistentVolumeClaim
	ServiceName          string
	PodManagementPolicy  v1.PodManagementPolicyType
	UpdateStrategy       v1.StatefulSetUpdateStrategy
	RevisionHistoryLimit *int32
}

// StatefulSetColumns returns kubernetes stateful set fields as Osquery table columns.
func StatefulSetColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&statefulSet{})
}

// StatefulSetsGenerate generates the kubernetes stateful sets as Osquery table data.
func StatefulSetsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		sss, err := k8s.GetClient().AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, ss := range sss.Items {
			item := &statefulSet{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(ss.ObjectMeta),
				CommonPodFields:        k8s.GetCommonPodFields(ss.Spec.Template.Spec),
				StatefulSetStatus:      ss.Status,
				StatefulSetReplicas:    ss.Spec.Replicas,
				Selector:               ss.Spec.Selector,
				VolumeClaimTemplates:   ss.Spec.VolumeClaimTemplates,
				ServiceName:            ss.Spec.ServiceName,
				PodManagementPolicy:    ss.Spec.PodManagementPolicy,
				UpdateStrategy:         ss.Spec.UpdateStrategy,
				RevisionHistoryLimit:   ss.Spec.RevisionHistoryLimit,
			}
			results = append(results, k8s.ToMap(item))
		}

		if sss.Continue == "" {
			break
		}
		options.Continue = sss.Continue
	}

	return results, nil
}

type statefulSetContainer struct {
	k8s.CommonNamespacedFields
	k8s.CommonContainerFields
	StatefulSetName string
	ContainerType   string
}

// StatefulSetContainerColumns returns kubernetes stateful set container fields as Osquery table columns.
func StatefulSetContainerColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&statefulSetContainer{})
}

// StatefulSetContainersGenerate generates the kubernetes stateful set containers as Osquery table data.
func StatefulSetContainersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		sss, err := k8s.GetClient().AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, ss := range sss.Items {
			for _, c := range ss.Spec.Template.Spec.InitContainers {
				item := &statefulSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(ss.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					StatefulSetName:        ss.Name,
					ContainerType:          "init",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range ss.Spec.Template.Spec.Containers {
				item := &statefulSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(ss.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					StatefulSetName:        ss.Name,
					ContainerType:          "container",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range ss.Spec.Template.Spec.EphemeralContainers {
				item := &statefulSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(ss.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonEphemeralContainerFields(c),
					StatefulSetName:        ss.Name,
					ContainerType:          "ephemeral",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if sss.Continue == "" {
			break
		}
		options.Continue = sss.Continue
	}

	return results, nil
}

type statefulSetVolume struct {
	k8s.CommonNamespacedFields
	k8s.CommonVolumeFields
	StatefulSetName string
}

// StatefulSetVolumeColumns returns kubernetes stateful set volume fields as Osquery table columns.
func StatefulSetVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&statefulSetVolume{})
}

// StatefulSetVolumesGenerate generates the kubernetes stateful set volumes as Osquery table data.
func StatefulSetVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		sss, err := k8s.GetClient().AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, ss := range sss.Items {
			for _, v := range ss.Spec.Template.Spec.Volumes {
				item := &statefulSetVolume{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(ss.ObjectMeta),
					CommonVolumeFields:     k8s.GetCommonVolumeFields(v),
					StatefulSetName:        ss.Name,
				}
				item.Name = v.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if sss.Continue == "" {
			break
		}
		options.Continue = sss.Continue
	}

	return results, nil
}
