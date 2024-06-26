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

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestMakeKey(t *testing.T) {
	assert.Equal(t, "host_ip", makeKey("HostIP"))
	assert.Equal(t, "host_ips", makeKey("HostIPs"))
	assert.Equal(t, "iscsi_interface", makeKey("ISCSIInterface"))
}

func TestGetSchema(t *testing.T) {
	assert.Equal(t, []table.ColumnDefinition{
		table.TextColumn("se_linux_options_user"),
		table.TextColumn("se_linux_options_role"),
		table.TextColumn("se_linux_options_type"),
		table.TextColumn("se_linux_options_level"),
		table.TextColumn("windows_options_gmsa_credential_spec_name"),
		table.TextColumn("windows_options_gmsa_credential_spec"),
		table.TextColumn("windows_options_run_as_user_name"),
		table.TextColumn("seccomp_profile_type"),
		table.TextColumn("seccomp_profile_localhost_profile"),
		table.BigIntColumn("run_as_user"),
		table.BigIntColumn("run_as_group"),
		table.IntegerColumn("run_as_non_root"),
		table.TextColumn("supplemental_groups"),
		table.BigIntColumn("fs_group"),
		table.TextColumn("sysctls"),
		table.TextColumn("fs_group_change_policy"),
		table.TextColumn("node_affinity"),
		table.TextColumn("pod_affinity"),
		table.TextColumn("pod_anti_affinity"),
		table.TextColumn("dns_config_nameservers"),
		table.TextColumn("dns_config_searches"),
		table.TextColumn("dns_config_options"),
		table.TextColumn("node_selector"),
		table.TextColumn("restart_policy"),
		table.BigIntColumn("termination_grace_period_seconds"),
		table.BigIntColumn("active_deadline_seconds"),
		table.TextColumn("dns_policy"),
		table.TextColumn("service_account_name"),
		table.IntegerColumn("automount_service_account_token"),
		table.TextColumn("node_name"),
		table.IntegerColumn("host_network"),
		table.IntegerColumn("host_pid"),
		table.IntegerColumn("host_ipc"),
		table.IntegerColumn("share_process_namespace"),
		table.TextColumn("image_pull_secrets"),
		table.TextColumn("hostname"),
		table.TextColumn("subdomain"),
		table.TextColumn("scheduler_name"),
		table.TextColumn("tolerations"),
		table.TextColumn("host_aliases"),
		table.TextColumn("priority_class_name"),
		table.IntegerColumn("priority"),
		table.TextColumn("readiness_gates"),
		table.TextColumn("runtime_class_name"),
		table.IntegerColumn("enable_service_links"),
		table.TextColumn("preemption_policy"),
		table.TextColumn("overhead"),
		table.TextColumn("topology_spread_constraints"),
		table.IntegerColumn("set_hostname_as_fqdn"),
	}, GetSchema(CommonPodFields{}))
}

func TestToMap(t *testing.T) {
	i32 := int32(456)
	i64 := int64(123)
	b := bool(true)
	s := string("s123")
	assert.Equal(t,
		map[string]string{
			"active_deadline_seconds":          "123",
			"automount_service_account_token":  "1",
			"dns_config_nameservers":           "[\"\"]",
			"dns_config_options":               "[{},{},{}]",
			"dns_config_searches":              "[\"\",\"\"]",
			"dns_policy":                       "ClusterFirst",
			"enable_service_links":             "1",
			"fs_group":                         "123",
			"fs_group_change_policy":           "s123",
			"host_aliases":                     "[{}]",
			"host_ipc":                         "1",
			"host_network":                     "1",
			"host_pid":                         "1",
			"hostname":                         "h123",
			"image_pull_secrets":               "[{},{},{}]",
			"node_affinity":                    "{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":null}}",
			"node_name":                        "n123",
			"node_selector":                    "{}",
			"overhead":                         "{}",
			"priority":                         "456",
			"priority_class_name":              "p123",
			"readiness_gates":                  "[]",
			"restart_policy":                   "Always",
			"run_as_group":                     "123",
			"run_as_non_root":                  "1",
			"run_as_user":                      "123",
			"runtime_class_name":               "s123",
			"scheduler_name":                   "sn123",
			"se_linux_options_role":            "r123",
			"se_linux_options_type":            "t123",
			"se_linux_options_user":            "u123",
			"seccomp_profile_type":             "Unconfined",
			"service_account_name":             "s123",
			"set_hostname_as_fqdn":             "1",
			"share_process_namespace":          "1",
			"subdomain":                        "sub123",
			"supplemental_groups":              "[0]",
			"sysctls":                          "[{\"name\":\"n1\",\"value\":\"v1\"}]",
			"termination_grace_period_seconds": "123",
			"tolerations":                      "[{},{}]",
			"topology_spread_constraints":      "[{\"maxSkew\":0,\"topologyKey\":\"\",\"whenUnsatisfiable\":\"\"},{\"maxSkew\":0,\"topologyKey\":\"\",\"whenUnsatisfiable\":\"\"},{\"maxSkew\":0,\"topologyKey\":\"\",\"whenUnsatisfiable\":\"\"},{\"maxSkew\":0,\"topologyKey\":\"\",\"whenUnsatisfiable\":\"\"}]",
		},
		ToMap(CommonPodFields{
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
			AffinityFields: AffinityFields{
				NodeAffinity:    &v1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{}},
				PodAffinity:     nil,
				PodAntiAffinity: nil,
			},
			DNSConfigFields: DNSConfigFields{
				DNSConfigNameservers: make([]string, 1),
				DNSConfigSearches:    make([]string, 2),
				DNSConfigOptions:     make([]v1.PodDNSConfigOption, 3),
			},
			PodSecurityContextFields: PodSecurityContextFields{
				CommonSecurityContextFields: CommonSecurityContextFields{
					SELinuxOptionsFields: SELinuxOptionsFields{
						SELinuxOptionsUser:  "u123",
						SELinuxOptionsRole:  "r123",
						SELinuxOptionsType:  "t123",
						SELinuxOptionsLevel: "",
					},
					WindowsOptionsFields: WindowsOptionsFields{},
					SeccompProfileFields: SeccompProfileFields{
						SeccompProfileType:             v1.SeccompProfileTypeUnconfined,
						SeccompProfileLocalhostProfile: nil,
					},
					RunAsUser:    &i64,
					RunAsGroup:   &i64,
					RunAsNonRoot: &b,
				},
				FSGroup:             &i64,
				FSGroupChangePolicy: (*v1.PodFSGroupChangePolicy)(&s),
				Sysctls:             []v1.Sysctl{{Name: "n1", Value: "v1"}},
				SupplementalGroups:  make([]int64, 1),
			},
		}))
}
