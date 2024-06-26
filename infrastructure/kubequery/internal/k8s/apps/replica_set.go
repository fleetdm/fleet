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

type replicaSet struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.ReplicaSetStatus
	ReplicaSetReplicas *int32
	MinReadySeconds    int32
	Selector           *metav1.LabelSelector
}

// ReplicaSetColumns returns kubernetes replica set fields as Osquery table columns.
func ReplicaSetColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&replicaSet{})
}

// ReplicaSetsGenerate generates the kubernetes replica sets as Osquery table data.
func ReplicaSetsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		rss, err := k8s.GetClient().AppsV1().ReplicaSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, rs := range rss.Items {
			item := &replicaSet{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(rs.ObjectMeta),
				CommonPodFields:        k8s.GetCommonPodFields(rs.Spec.Template.Spec),
				ReplicaSetStatus:       rs.Status,
				ReplicaSetReplicas:     rs.Spec.Replicas,
				MinReadySeconds:        rs.Spec.MinReadySeconds,
				Selector:               rs.Spec.Selector,
			}
			results = append(results, k8s.ToMap(item))
		}

		if rss.Continue == "" {
			break
		}
		options.Continue = rss.Continue
	}

	return results, nil
}

type replicaSetContainer struct {
	k8s.CommonNamespacedFields
	k8s.CommonContainerFields
	ReplicaSetName string
	ContainerType  string
}

// ReplicaSetContainerColumns returns kubernetes replica set container fields as Osquery table columns.
func ReplicaSetContainerColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&replicaSetContainer{})
}

// ReplicaSetContainersGenerate generates the kubernetes replica set containers as Osquery table data.
func ReplicaSetContainersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		rss, err := k8s.GetClient().AppsV1().ReplicaSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, rs := range rss.Items {
			for _, c := range rs.Spec.Template.Spec.InitContainers {
				item := &replicaSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(rs.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					ReplicaSetName:         rs.Name,
					ContainerType:          "init",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range rs.Spec.Template.Spec.Containers {
				item := &replicaSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(rs.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonContainerFields(c),
					ReplicaSetName:         rs.Name,
					ContainerType:          "container",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
			for _, c := range rs.Spec.Template.Spec.EphemeralContainers {
				item := &replicaSetContainer{
					CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(rs.ObjectMeta, c.Name),
					CommonContainerFields:  k8s.GetCommonEphemeralContainerFields(c),
					ReplicaSetName:         rs.Name,
					ContainerType:          "ephemeral",
				}
				item.Name = c.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if rss.Continue == "" {
			break
		}
		options.Continue = rss.Continue
	}

	return results, nil
}

type replicaSetVolume struct {
	k8s.CommonNamespacedFields
	k8s.CommonVolumeFields
	ReplicaSetName string
}

// ReplicaSetVolumeColumns returns kubernetes replica set volume fields as Osquery table columns.
func ReplicaSetVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&replicaSetVolume{})
}

// ReplicaSetVolumesGenerate generates the kubernetes replica set volumes as Osquery table data.
func ReplicaSetVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		rss, err := k8s.GetClient().AppsV1().ReplicaSets(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, rs := range rss.Items {
			for _, v := range rs.Spec.Template.Spec.Volumes {
				item := &replicaSetVolume{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(rs.ObjectMeta),
					CommonVolumeFields:     k8s.GetCommonVolumeFields(v),
					ReplicaSetName:         rs.Name,
				}
				item.Name = v.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if rss.Continue == "" {
			break
		}
		options.Continue = rss.Continue
	}

	return results, nil
}
