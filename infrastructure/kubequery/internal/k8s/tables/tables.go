/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package tables

import (
	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s/admissionregistration"
	"github.com/Uptycs/kubequery/internal/k8s/apps"
	"github.com/Uptycs/kubequery/internal/k8s/autoscaling"
	"github.com/Uptycs/kubequery/internal/k8s/batch"
	"github.com/Uptycs/kubequery/internal/k8s/core"
	"github.com/Uptycs/kubequery/internal/k8s/discovery"
	"github.com/Uptycs/kubequery/internal/k8s/event"
	"github.com/Uptycs/kubequery/internal/k8s/networking"
	"github.com/Uptycs/kubequery/internal/k8s/policy"
	"github.com/Uptycs/kubequery/internal/k8s/rbac"
	"github.com/Uptycs/kubequery/internal/k8s/storage"
)

// Table structure holds Osquery extension table definition.
type Table struct {
	Name    string
	Columns []table.ColumnDefinition
	GenFunc table.GenerateFunc
}

// GetTables returns the definition of all the tables supported by this extension.
func GetTables() []Table {
	return []Table{
		// Admission Registration
		{"kubernetes_mutating_webhooks", admissionregistration.MutatingWebhookColumns(), admissionregistration.MutatingWebhooksGenerate},
		{"kubernetes_validating_webhooks", admissionregistration.ValidatingWebhookColumns(), admissionregistration.ValidatingWebhooksGenerate},

		// Apps
		{"kubernetes_daemon_sets", apps.DaemonSetColumns(), apps.DaemonSetsGenerate},
		{"kubernetes_daemon_set_containers", apps.DaemonSetContainerColumns(), apps.DaemonSetContainersGenerate},
		{"kubernetes_daemon_set_volumes", apps.DaemonSetVolumeColumns(), apps.DaemonSetVolumesGenerate},
		{"kubernetes_deployments", apps.DeploymentColumns(), apps.DeploymentsGenerate},
		{"kubernetes_deployments_containers", apps.DeploymentContainerColumns(), apps.DeploymentContainersGenerate},
		{"kubernetes_deployments_volumes", apps.DeploymentVolumeColumns(), apps.DeploymentVolumesGenerate},
		{"kubernetes_replica_sets", apps.ReplicaSetColumns(), apps.ReplicaSetsGenerate},
		{"kubernetes_replica_set_containers", apps.ReplicaSetContainerColumns(), apps.ReplicaSetContainersGenerate},
		{"kubernetes_replica_set_volumes", apps.ReplicaSetVolumeColumns(), apps.ReplicaSetVolumesGenerate},
		{"kubernetes_stateful_sets", apps.StatefulSetColumns(), apps.StatefulSetsGenerate},
		{"kubernetes_stateful_set_containers", apps.StatefulSetContainerColumns(), apps.StatefulSetContainersGenerate},
		{"kubernetes_stateful_set_volumes", apps.StatefulSetVolumeColumns(), apps.StatefulSetVolumesGenerate},

		// Autoscaling
		{"kubernetes_horizontal_pod_autoscalers", autoscaling.HorizontalPodAutoscalersColumns(), autoscaling.HorizontalPodAutoscalerGenerate},

		// Batch
		{"kubernetes_cron_jobs", batch.CronJobColumns(), batch.CronJobsGenerate},
		{"kubernetes_jobs", batch.JobColumns(), batch.JobsGenerate},

		// Core
		{"kubernetes_component_statuses", core.ComponentStatusColumns(), core.ComponentStatusesGenerate},
		{"kubernetes_config_maps", core.ConfigMapColumns(), core.ConfigMapsGenerate},
		{"kubernetes_endpoint_subsets", core.EndpointSubsetColumns(), core.EndpointSubsetsGenerate},
		{"kubernetes_limit_ranges", core.LimitRangeColumns(), core.LimitRangesGenerate},
		{"kubernetes_namespaces", core.NamespaceColumns(), core.NamespacesGenerate},
		{"kubernetes_nodes", core.NodeColumns(), core.NodesGenerate},
		{"kubernetes_persistent_volume_claims", core.PersistentVolumeClaimColumns(), core.PersistentVolumeClaimsGenerate},
		{"kubernetes_persistent_volumes", core.PersistentVolumeColumns(), core.PersistentVolumesGenerate},
		{"kubernetes_pod_templates", core.PodTemplateColumns(), core.PodTemplatesGenerate},
		{"kubernetes_pod_template_containers", core.PodTemplateContainerColumns(), core.PodTemplateContainersGenerate},
		{"kubernetes_pod_templates_volumes", core.PodTemplateVolumeColumns(), core.PodTemplateVolumesGenerate},
		{"kubernetes_pods", core.PodColumns(), core.PodsGenerate},
		{"kubernetes_pod_containers", core.PodContainerColumns(), core.PodContainersGenerate},
		{"kubernetes_pod_volumes", core.PodVolumeColumns(), core.PodVolumesGenerate},
		{"kubernetes_resource_quotas", core.ResourceQuotaColumns(), core.ResourceQuotasGenerate},
		{"kubernetes_secrets", core.SecretColumns(), core.SecretsGenerate},
		{"kubernetes_service_accounts", core.ServiceAccountColumns(), core.ServiceAccountsGenerate},
		{"kubernetes_services", core.ServiceColumns(), core.ServicesGenerate},

		// Discovery
		{"kubernetes_api_resources", discovery.APIResourceColumns(), discovery.APIResourcesGenerate},
		{"kubernetes_info", discovery.InfoColumns(), discovery.InfoGenerate},

		// Event
		{"kubernetes_events", event.Columns(), event.Generate},

		// Networking
		{"kubernetes_ingress_classes", networking.IngressClassColumns(), networking.IngressClassesGenerate},
		{"kubernetes_ingresses", networking.IngressColumns(), networking.IngressesGenerate},
		{"kubernetes_network_policies", networking.NetworkPolicyColumns(), networking.NetworkPoliciesGenerate},

		// Policy
		{"kubernetes_pod_disruption_budgets", policy.PodDisruptionBudgetColumns(), policy.PodDisruptionBudgetsGenerate},
		{"kubernetes_pod_security_policies", policy.PodSecurityPolicyColumns(), policy.PodSecurityPoliciesGenerate},

		// RBAC
		{"kubernetes_cluster_role_binding_subjects", rbac.ClusterRoleBindingSubjectColumns(), rbac.ClusterRoleBindingSubjectsGenerate},
		{"kubernetes_cluster_role_policy_rules", rbac.ClusterRolePolicyRuleColumns(), rbac.ClusterRolePolicyRulesGenerate},
		{"kubernetes_role_binding_subjects", rbac.RoleBindingSubjectColumns(), rbac.RoleBindingSubjectsGenerate},
		{"kubernetes_role_policy_rules", rbac.RolePolicyRuleColumns(), rbac.RolePolicyRulesGenerate},

		// Storage
		{"kubernetes_csi_drivers", storage.CSIDriverColumns(), storage.CSIDriversGenerate},
		{"kubernetes_csi_node_drivers", storage.CSINodeDriverColumns(), storage.CSINodeDriversGenerate},
		// {"kubernetes_storage_capacities", storage.CSIStorageCapacityColumns(), storage.CSIStorageCapacitiesGenerate},
		{"kubernetes_storage_classes", storage.SCClassColumns(), storage.SCClassesGenerate},
		{"kubernetes_volume_attachments", storage.VolumeAttachmentColumns(), storage.VolumeAttachmentsGenerate},
	}
}
