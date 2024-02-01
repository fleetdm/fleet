/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetCommonFields(t *testing.T) {
	meta := metav1.ObjectMeta{
		Name:                       "n123",
		GenerateName:               "g123",
		Namespace:                  "kube-system",
		SelfLink:                   "/",
		UID:                        types.UID("u123"),
		ResourceVersion:            "r123",
		Generation:                 1,
		CreationTimestamp:          metav1.Time{},
		DeletionGracePeriodSeconds: nil,
		Labels:                     map[string]string{"a": "b"},
		ClusterName:                "",
	}
	assert.Equal(t, GetCommonFields(meta), CommonFields{
		UID:               meta.UID,
		Name:              meta.Name,
		ClusterName:       "cluster-name",
		ClusterUID:        types.UID("uid"),
		CreationTimestamp: meta.CreationTimestamp,
		Labels:            meta.Labels,
		Annotations:       meta.Annotations,
	}, "Common fields should match")
}

func TestGetNamespaceCommonFields(t *testing.T) {
	meta := metav1.ObjectMeta{
		Name:                       "n123",
		GenerateName:               "g123",
		Namespace:                  "kube-system",
		SelfLink:                   "/",
		UID:                        types.UID("u123"),
		ResourceVersion:            "r123",
		Generation:                 1,
		CreationTimestamp:          metav1.Time{},
		DeletionGracePeriodSeconds: nil,
		Annotations:                map[string]string{"a": "b"},
		ClusterName:                "",
	}
	assert.Equal(t, GetCommonNamespacedFields(meta), CommonNamespacedFields{
		UID:               meta.UID,
		Name:              meta.Name,
		Namespace:         meta.Namespace,
		ClusterName:       "cluster-name",
		ClusterUID:        types.UID("uid"),
		CreationTimestamp: meta.CreationTimestamp,
		Labels:            meta.Labels,
		Annotations:       meta.Annotations,
	}, "Common namespace fields should match")
}

func TestGetCommonPodFields(t *testing.T) {
	i32 := int32(456)
	i64 := int64(123)
	b := bool(true)
	s := string("s123")
	pod := v1.PodSpec{
		RestartPolicy:                 v1.RestartPolicyAlways,
		TerminationGracePeriodSeconds: &i64,
		ActiveDeadlineSeconds:         &i64,
		DNSPolicy:                     v1.DNSClusterFirst,
		ServiceAccountName:            "s123",
		AutomountServiceAccountToken:  &b,
		NodeSelector:                  make(map[string]string),
		NodeName:                      "n123",
		HostNetwork:                   true,
		HostPID:                       true,
		HostIPC:                       true,
		ShareProcessNamespace:         &b,
		ImagePullSecrets:              make([]v1.LocalObjectReference, 3),
		Hostname:                      "h123",
		Subdomain:                     "sub123",
		Affinity:                      &v1.Affinity{NodeAffinity: &v1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{}}},
		SchedulerName:                 "sn123",
		Tolerations:                   make([]v1.Toleration, 2),
		HostAliases:                   make([]v1.HostAlias, 1),
		PriorityClassName:             "p123",
		Priority:                      &i32,
		ReadinessGates:                []v1.PodReadinessGate{},
		RuntimeClassName:              &s,
		EnableServiceLinks:            &b,
		PreemptionPolicy:              nil,
		Overhead:                      make(v1.ResourceList),
		TopologySpreadConstraints:     make([]v1.TopologySpreadConstraint, 4),
		SetHostnameAsFQDN:             &b,
		DNSConfig: &v1.PodDNSConfig{
			Nameservers: make([]string, 1),
			Searches:    make([]string, 2),
			Options:     make([]v1.PodDNSConfigOption, 3),
		},
		SecurityContext: &v1.PodSecurityContext{
			RunAsUser:           &i64,
			RunAsGroup:          &i64,
			RunAsNonRoot:        &b,
			FSGroup:             &i64,
			FSGroupChangePolicy: (*v1.PodFSGroupChangePolicy)(&s),
			Sysctls:             []v1.Sysctl{{Name: "n1", Value: "v1"}},
			SELinuxOptions:      &v1.SELinuxOptions{User: "u123", Role: "r123", Type: "t123", Level: "l123"},
			SupplementalGroups:  make([]int64, 1),
			SeccompProfile:      &v1.SeccompProfile{Type: "t123"},
		},
	}
	assert.Equal(t, GetCommonPodFields(pod), CommonPodFields{
		PodSecurityContextFields: PodSecurityContextFields{
			CommonSecurityContextFields: CommonSecurityContextFields{
				SELinuxOptionsFields: SELinuxOptionsFields{
					SELinuxOptionsUser:  pod.SecurityContext.SELinuxOptions.User,
					SELinuxOptionsRole:  pod.SecurityContext.SELinuxOptions.Role,
					SELinuxOptionsType:  pod.SecurityContext.SELinuxOptions.Type,
					SELinuxOptionsLevel: pod.SecurityContext.SELinuxOptions.Level,
				},
				WindowsOptionsFields: WindowsOptionsFields{
					WindowsOptionsGMSACredentialSpecName: nil,
					WindowsOptionsGMSACredentialSpec:     nil,
					WindowsOptionsRunAsUserName:          nil,
				},
				SeccompProfileFields: SeccompProfileFields{
					SeccompProfileType:             pod.SecurityContext.SeccompProfile.Type,
					SeccompProfileLocalhostProfile: pod.SecurityContext.SeccompProfile.LocalhostProfile,
				},
				RunAsUser:    pod.SecurityContext.RunAsUser,
				RunAsGroup:   pod.SecurityContext.RunAsGroup,
				RunAsNonRoot: pod.SecurityContext.RunAsNonRoot,
			},
			SupplementalGroups:  pod.SecurityContext.SupplementalGroups,
			FSGroup:             pod.SecurityContext.FSGroup,
			Sysctls:             pod.SecurityContext.Sysctls,
			FSGroupChangePolicy: pod.SecurityContext.FSGroupChangePolicy,
		},
		DNSConfigFields: DNSConfigFields{
			DNSConfigNameservers: pod.DNSConfig.Nameservers,
			DNSConfigSearches:    pod.DNSConfig.Searches,
			DNSConfigOptions:     pod.DNSConfig.Options,
		},
		AffinityFields: AffinityFields{
			NodeAffinity:    pod.Affinity.NodeAffinity,
			PodAffinity:     pod.Affinity.PodAffinity,
			PodAntiAffinity: pod.Affinity.PodAntiAffinity,
		},
		NodeSelector:                  pod.NodeSelector,
		RestartPolicy:                 pod.RestartPolicy,
		TerminationGracePeriodSeconds: pod.TerminationGracePeriodSeconds,
		ActiveDeadlineSeconds:         pod.ActiveDeadlineSeconds,
		DNSPolicy:                     pod.DNSPolicy,
		ServiceAccountName:            pod.ServiceAccountName,
		AutomountServiceAccountToken:  pod.AutomountServiceAccountToken,
		NodeName:                      pod.NodeName,
		HostNetwork:                   pod.HostNetwork,
		HostPID:                       pod.HostPID,
		HostIPC:                       pod.HostIPC,
		ShareProcessNamespace:         pod.ShareProcessNamespace,
		ImagePullSecrets:              pod.ImagePullSecrets,
		Hostname:                      pod.Hostname,
		Subdomain:                     pod.Subdomain,
		SchedulerName:                 pod.SchedulerName,
		Tolerations:                   pod.Tolerations,
		HostAliases:                   pod.HostAliases,
		PriorityClassName:             pod.PriorityClassName,
		Priority:                      pod.Priority,
		ReadinessGates:                pod.ReadinessGates,
		RuntimeClassName:              pod.RuntimeClassName,
		EnableServiceLinks:            pod.EnableServiceLinks,
		PreemptionPolicy:              pod.PreemptionPolicy,
		Overhead:                      pod.Overhead,
		TopologySpreadConstraints:     pod.TopologySpreadConstraints,
		SetHostnameAsFQDN:             pod.SetHostnameAsFQDN,
	}, "Common pod fields should match")
}

func TestGetCommonContainerFields(t *testing.T) {
	i64 := int64(456)
	b := bool(true)
	s := string("str123")
	c := v1.Container{
		Name:                     "n123",
		Image:                    "i123",
		Command:                  []string{"c123"},
		Args:                     []string{"a1", "a2"},
		WorkingDir:               "w123",
		Ports:                    make([]v1.ContainerPort, 1),
		EnvFrom:                  []v1.EnvFromSource{{Prefix: "p123"}},
		Env:                      []v1.EnvVar{{Name: "n1", Value: "v1"}},
		Resources:                v1.ResourceRequirements{Limits: v1.ResourceList{}},
		VolumeMounts:             make([]v1.VolumeMount, 2),
		VolumeDevices:            []v1.VolumeDevice{{Name: "vn1"}},
		LivenessProbe:            &v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"curl"}}}},
		ReadinessProbe:           &v1.Probe{},
		StartupProbe:             nil,
		Lifecycle:                &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: []string{"curl"}}}},
		TerminationMessagePath:   "t123",
		TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		ImagePullPolicy:          v1.PullAlways,
		Stdin:                    true,
		StdinOnce:                false,
		TTY:                      true,
		SecurityContext: &v1.SecurityContext{
			Capabilities:             &v1.Capabilities{Add: []v1.Capability{"a"}, Drop: []v1.Capability{"b", "c"}},
			Privileged:               &b,
			RunAsUser:                &i64,
			RunAsGroup:               nil,
			RunAsNonRoot:             &b,
			ReadOnlyRootFilesystem:   nil,
			AllowPrivilegeEscalation: &b,
			ProcMount:                nil,
			SELinuxOptions: &v1.SELinuxOptions{
				User:  "u123",
				Role:  "r123",
				Type:  "t123",
				Level: "",
			},
			SeccompProfile: &v1.SeccompProfile{
				Type:             v1.SeccompProfileType("abc"),
				LocalhostProfile: &s,
			},
			WindowsOptions: nil,
		},
	}
	assert.Equal(t, GetCommonContainerFields(c), CommonContainerFields{
		SecurityContextFields: SecurityContextFields{
			CommonSecurityContextFields: CommonSecurityContextFields{
				SELinuxOptionsFields: SELinuxOptionsFields{
					SELinuxOptionsUser:  c.SecurityContext.SELinuxOptions.User,
					SELinuxOptionsRole:  c.SecurityContext.SELinuxOptions.Role,
					SELinuxOptionsType:  c.SecurityContext.SELinuxOptions.Type,
					SELinuxOptionsLevel: c.SecurityContext.SELinuxOptions.Level,
				},
				WindowsOptionsFields: WindowsOptionsFields{
					WindowsOptionsGMSACredentialSpecName: nil,
					WindowsOptionsGMSACredentialSpec:     nil,
					WindowsOptionsRunAsUserName:          nil,
				},
				SeccompProfileFields: SeccompProfileFields{
					SeccompProfileType:             c.SecurityContext.SeccompProfile.Type,
					SeccompProfileLocalhostProfile: c.SecurityContext.SeccompProfile.LocalhostProfile,
				},
				RunAsUser:    c.SecurityContext.RunAsUser,
				RunAsGroup:   c.SecurityContext.RunAsGroup,
				RunAsNonRoot: c.SecurityContext.RunAsNonRoot,
			},
			CapabilitiesAdd:          c.SecurityContext.Capabilities.Add,
			CapabilitiesDrop:         c.SecurityContext.Capabilities.Drop,
			Privileged:               c.SecurityContext.Privileged,
			ReadOnlyRootFilesystem:   c.SecurityContext.ReadOnlyRootFilesystem,
			AllowPrivilegeEscalation: c.SecurityContext.AllowPrivilegeEscalation,
			ProcMount:                c.SecurityContext.ProcMount,
		},
		TargetContainerName:      "",
		Image:                    c.Image,
		Command:                  c.Command,
		Args:                     c.Args,
		WorkingDir:               c.WorkingDir,
		Ports:                    c.Ports,
		EnvFrom:                  c.EnvFrom,
		Env:                      c.Env,
		ResourceLimits:           c.Resources.Limits,
		ResourceRequests:         c.Resources.Requests,
		VolumeMounts:             c.VolumeMounts,
		VolumeDevices:            c.VolumeDevices,
		LivenessProbe:            c.LivenessProbe,
		ReadinessProbe:           c.ReadinessProbe,
		StartupProbe:             c.StartupProbe,
		Lifecycle:                c.Lifecycle,
		TerminationMessagePath:   c.TerminationMessagePath,
		TerminationMessagePolicy: c.TerminationMessagePolicy,
		ImagePullPolicy:          c.ImagePullPolicy,
		Stdin:                    c.Stdin,
		StdinOnce:                c.StdinOnce,
		TTY:                      c.TTY,
	}, "Common container fields should match")
}

func TestGetCommonEphemeralContainerFields(t *testing.T) {
	i64 := int64(456)
	b := bool(true)
	s := string("str123")
	c := v1.EphemeralContainer{
		TargetContainerName: "t123",
		EphemeralContainerCommon: v1.EphemeralContainerCommon{
			Name:                     "n123",
			Image:                    "i123",
			Command:                  []string{"c123"},
			Args:                     []string{"a1", "a2"},
			WorkingDir:               "w123",
			Ports:                    make([]v1.ContainerPort, 1),
			EnvFrom:                  []v1.EnvFromSource{{Prefix: "p123"}},
			Env:                      []v1.EnvVar{{Name: "n1", Value: "v1"}},
			Resources:                v1.ResourceRequirements{Limits: v1.ResourceList{}},
			VolumeMounts:             make([]v1.VolumeMount, 2),
			VolumeDevices:            []v1.VolumeDevice{{Name: "vn1"}},
			LivenessProbe:            &v1.Probe{Handler: v1.Handler{Exec: &v1.ExecAction{Command: []string{"curl"}}}},
			ReadinessProbe:           &v1.Probe{},
			StartupProbe:             nil,
			Lifecycle:                &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: []string{"curl"}}}},
			TerminationMessagePath:   "t123",
			TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
			ImagePullPolicy:          v1.PullAlways,
			Stdin:                    true,
			StdinOnce:                false,
			TTY:                      true,
			SecurityContext: &v1.SecurityContext{
				Capabilities:             &v1.Capabilities{Add: []v1.Capability{"a"}, Drop: []v1.Capability{"b", "c"}},
				Privileged:               &b,
				RunAsUser:                &i64,
				RunAsGroup:               nil,
				RunAsNonRoot:             &b,
				ReadOnlyRootFilesystem:   nil,
				AllowPrivilegeEscalation: &b,
				ProcMount:                nil,
				SELinuxOptions: &v1.SELinuxOptions{
					User:  "u123",
					Role:  "r123",
					Type:  "t123",
					Level: "",
				},
				SeccompProfile: &v1.SeccompProfile{
					Type:             v1.SeccompProfileType("abc"),
					LocalhostProfile: &s,
				},
				WindowsOptions: nil,
			},
		},
	}
	assert.Equal(t, GetCommonEphemeralContainerFields(c), CommonContainerFields{
		SecurityContextFields: SecurityContextFields{
			CommonSecurityContextFields: CommonSecurityContextFields{
				SELinuxOptionsFields: SELinuxOptionsFields{
					SELinuxOptionsUser:  c.SecurityContext.SELinuxOptions.User,
					SELinuxOptionsRole:  c.SecurityContext.SELinuxOptions.Role,
					SELinuxOptionsType:  c.SecurityContext.SELinuxOptions.Type,
					SELinuxOptionsLevel: c.SecurityContext.SELinuxOptions.Level,
				},
				WindowsOptionsFields: WindowsOptionsFields{
					WindowsOptionsGMSACredentialSpecName: nil,
					WindowsOptionsGMSACredentialSpec:     nil,
					WindowsOptionsRunAsUserName:          nil,
				},
				SeccompProfileFields: SeccompProfileFields{
					SeccompProfileType:             c.SecurityContext.SeccompProfile.Type,
					SeccompProfileLocalhostProfile: c.SecurityContext.SeccompProfile.LocalhostProfile,
				},
				RunAsUser:    c.SecurityContext.RunAsUser,
				RunAsGroup:   c.SecurityContext.RunAsGroup,
				RunAsNonRoot: c.SecurityContext.RunAsNonRoot,
			},
			CapabilitiesAdd:          c.SecurityContext.Capabilities.Add,
			CapabilitiesDrop:         c.SecurityContext.Capabilities.Drop,
			Privileged:               c.SecurityContext.Privileged,
			ReadOnlyRootFilesystem:   c.SecurityContext.ReadOnlyRootFilesystem,
			AllowPrivilegeEscalation: c.SecurityContext.AllowPrivilegeEscalation,
			ProcMount:                c.SecurityContext.ProcMount,
		},
		TargetContainerName:      c.TargetContainerName,
		Image:                    c.Image,
		Command:                  c.Command,
		Args:                     c.Args,
		WorkingDir:               c.WorkingDir,
		Ports:                    c.Ports,
		EnvFrom:                  c.EnvFrom,
		Env:                      c.Env,
		ResourceLimits:           c.Resources.Limits,
		ResourceRequests:         c.Resources.Requests,
		VolumeMounts:             c.VolumeMounts,
		VolumeDevices:            c.VolumeDevices,
		LivenessProbe:            c.LivenessProbe,
		ReadinessProbe:           c.ReadinessProbe,
		StartupProbe:             c.StartupProbe,
		Lifecycle:                c.Lifecycle,
		TerminationMessagePath:   c.TerminationMessagePath,
		TerminationMessagePolicy: c.TerminationMessagePolicy,
		ImagePullPolicy:          c.ImagePullPolicy,
		Stdin:                    c.Stdin,
		StdinOnce:                c.StdinOnce,
		TTY:                      c.TTY,
	}, "Common ephemeral container fields should match")
}

func TestGetCommonVolumeFields(t *testing.T) {
	v := v1.Volume{
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "p123",
				Type: nil,
			},
		},
	}
	assert.Equal(t, GetCommonVolumeFields(v), CommonVolumeFields{
		VolumeType:   "host_path",
		HostPathPath: v.HostPath.Path,
		HostPathType: v.HostPath.Type,
	}, "Common volume HostPath fields should match")

	v = v1.Volume{
		VolumeSource: v1.VolumeSource{
			GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{
				PDName:    "p123",
				FSType:    "gce",
				Partition: 123,
			},
		},
	}
	assert.Equal(t, GetCommonVolumeFields(v), CommonVolumeFields{
		VolumeType:                 "gce_persistent_disk",
		FSType:                     &v.GCEPersistentDisk.FSType,
		ReadOnly:                   &v.GCEPersistentDisk.ReadOnly,
		GCEPersistentDiskPDName:    v.GCEPersistentDisk.PDName,
		GCEPersistentDiskPartition: v.GCEPersistentDisk.Partition,
	}, "Common volume GCE fields should match")
}
