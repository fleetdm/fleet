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
	"strings"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pod struct {
	k8s.CommonNamespacedFields
	k8s.CommonPodFields
	v1.PodStatus
}

// PodColumns returns kubernetes pod fields as Osquery table columns.
func PodColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&pod{})
}

// PodsGenerate generates the kubernetes pods as Osquery table data.
func PodsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pods, err := k8s.GetClient().CoreV1().Pods(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, p := range pods.Items {
			item := &pod{
				CommonNamespacedFields: k8s.GetCommonNamespacedFields(p.ObjectMeta),
				CommonPodFields:        k8s.GetCommonPodFields(p.Spec),
				PodStatus:              p.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if pods.Continue == "" {
			break
		}
		options.Continue = pods.Continue
	}

	return results, nil
}

type podContainer struct {
	k8s.CommonNamespacedFields
	k8s.CommonContainerFields
	PodName              string
	ContainerType        string
	State                v1.ContainerState
	LastTerminationState v1.ContainerState
	Ready                bool
	RestartCount         int32
	ImageRepo            string
	ImageID              string
	ContainerID          string
	Started              *bool
}

// PodContainerColumns returns kubernetes pod container fields as Osquery table columns.
func PodContainerColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podContainer{})
}

func getImageRepo(id string) string {
	// docker.io/jaegertracing/jaeger-operator@sha256:5a3198179f7972028a29dd7fbf71ac7a21e0dbf46c85e8cc2c37e3b6a5ee26a4
	index := strings.LastIndex(id, "@")
	if index < 0 || index == len(id)-1 {
		return ""
	}
	return id[0:index]
}

func cleanID(id string) string {
	// containerd://4a8e3f149f24fb5d4429f4a38e86097e1aec3b6b174bb382a44c6706ad4406e1
	// docker.io/jaegertracing/jaeger-operator@sha256:5a3198179f7972028a29dd7fbf71ac7a21e0dbf46c85e8cc2c37e3b6a5ee26a4
	index := -1
	for _, s := range []string{"/", ":", "@"} {
		i := strings.LastIndex(id, s)
		if i > -1 && i > index {
			index = i
		}
	}

	if index < 0 || index == len(id)-1 {
		return id
	}
	return id[index+1:]
}

func updatePodContainerStatus(pc *podContainer, cs *v1.ContainerStatus) {
	if cs != nil {
		pc.State = cs.State
		pc.LastTerminationState = cs.LastTerminationState
		pc.Ready = cs.Ready
		pc.RestartCount = cs.RestartCount
		pc.ImageRepo = getImageRepo(cs.ImageID)
		pc.ImageID = cleanID(cs.ImageID)
		pc.ContainerID = cleanID(cs.ContainerID)
		pc.Started = cs.Started
	}
}

func createPodContainer(p v1.Pod, c v1.Container, cs *v1.ContainerStatus, containerType string) *podContainer {
	item := &podContainer{
		CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(p.ObjectMeta, c.Name),
		CommonContainerFields:  k8s.GetCommonContainerFields(c),
		PodName:                p.Name,
		ContainerType:          containerType,
	}
	item.Name = c.Name
	updatePodContainerStatus(item, cs)
	return item
}

func createPodEphemeralContainer(p v1.Pod, c v1.EphemeralContainer, cs *v1.ContainerStatus) *podContainer {
	item := &podContainer{
		CommonNamespacedFields: k8s.GetParentCommonNamespacedFields(p.ObjectMeta, c.Name),
		CommonContainerFields:  k8s.GetCommonEphemeralContainerFields(c),
		PodName:                p.Name,
		ContainerType:          "ephemeral",
	}
	item.Name = c.Name
	updatePodContainerStatus(item, cs)
	return item
}

// PodContainersGenerate generates the kubernetes pod containers as Osquery table data.
func PodContainersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pods, err := k8s.GetClient().CoreV1().Pods(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, p := range pods.Items {
			for i, c := range p.Spec.InitContainers {
				var cs *v1.ContainerStatus = nil
				if len(p.Status.InitContainerStatuses) > i {
					cs = &p.Status.InitContainerStatuses[i]
				}
				item := createPodContainer(p, c, cs, "init")
				results = append(results, k8s.ToMap(item))
			}
			for i, c := range p.Spec.Containers {
				var cs *v1.ContainerStatus = nil
				if len(p.Status.ContainerStatuses) > i {
					cs = &p.Status.ContainerStatuses[i]
				}
				item := createPodContainer(p, c, cs, "container")
				results = append(results, k8s.ToMap(item))
			}
			for i, c := range p.Spec.EphemeralContainers {
				var cs *v1.ContainerStatus = nil
				if len(p.Status.EphemeralContainerStatuses) > i {
					cs = &p.Status.EphemeralContainerStatuses[i]
				}
				item := createPodEphemeralContainer(p, c, cs)
				results = append(results, k8s.ToMap(item))
			}
		}

		if pods.Continue == "" {
			break
		}
		options.Continue = pods.Continue
	}

	return results, nil
}

type podVolume struct {
	k8s.CommonNamespacedFields
	k8s.CommonVolumeFields
	PodName string
}

// PodVolumeColumns returns kubernetes pod volume fields as Osquery table columns.
func PodVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&podVolume{})
}

// PodVolumesGenerate generates the kubernetes pod volumes as Osquery table data.
func PodVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pods, err := k8s.GetClient().CoreV1().Pods(metav1.NamespaceAll).List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, p := range pods.Items {
			for _, v := range p.Spec.Volumes {
				item := &podVolume{
					CommonNamespacedFields: k8s.GetCommonNamespacedFields(p.ObjectMeta),
					CommonVolumeFields:     k8s.GetCommonVolumeFields(v),
					PodName:                p.Name,
				}
				item.Name = v.Name
				results = append(results, k8s.ToMap(item))
			}
		}

		if pods.Continue == "" {
			break
		}
		options.Continue = pods.Continue
	}

	return results, nil
}
