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
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CommonFields contains fields common to most tables. Contents are derived from kubernetes ObjectMeta.
// This is used for kubernetes resources that are not namespaced.
type CommonFields struct {
	UID               types.UID
	ClusterName       string
	ClusterUID        types.UID
	Name              string
	CreationTimestamp metav1.Time
	Labels            map[string]string
	Annotations       map[string]string
}

// GetCommonFields returns CommonFields struct from the provided kubernetes ObjectMeta.
func GetCommonFields(obj metav1.ObjectMeta) CommonFields {
	return CommonFields{
		UID:               obj.UID,
		ClusterName:       GetClusterName(),
		ClusterUID:        GetClusterUID(),
		Name:              obj.Name,
		CreationTimestamp: obj.CreationTimestamp,
		Labels:            obj.Labels,
		Annotations:       obj.Annotations,
	}
}

// CommonNamespacedFields contains fields common to most tables. Contents are derived from kubernetes ObjectMeta.
// This is used for kubernetes resources that are namespaced.
type CommonNamespacedFields struct {
	UID               types.UID
	ClusterName       string
	ClusterUID        types.UID
	Name              string
	Namespace         string
	CreationTimestamp metav1.Time
	Labels            map[string]string
	Annotations       map[string]string
}

// GetCommonNamespacedFields returns CommonNamespacedFields struct from the provided kubernetes ObjectMeta.
func GetCommonNamespacedFields(obj metav1.ObjectMeta) CommonNamespacedFields {
	return CommonNamespacedFields{
		UID:               obj.UID,
		ClusterName:       GetClusterName(),
		ClusterUID:        GetClusterUID(),
		Name:              obj.Name,
		Namespace:         obj.Namespace,
		CreationTimestamp: obj.CreationTimestamp,
		Labels:            obj.Labels,
		Annotations:       obj.Annotations,
	}
}

// GetParentCommonNamespacedFields returns CommonNamespacedFields struct from the parent ObjectMeta creating a UID using parent UID + provided name.
func GetParentCommonNamespacedFields(parent metav1.ObjectMeta, name string) CommonNamespacedFields {
	uid := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(string(parent.UID)+name)).String()
	return CommonNamespacedFields{
		UID:               types.UID(uid),
		ClusterName:       GetClusterName(),
		ClusterUID:        GetClusterUID(),
		Name:              name,
		Namespace:         parent.Namespace,
		CreationTimestamp: parent.CreationTimestamp,
		Labels:            parent.Labels,
		Annotations:       parent.Annotations,
	}
}

// SELinuxOptionsFields contains SELinux options as a flat structure.
type SELinuxOptionsFields struct {
	SELinuxOptionsUser  string
	SELinuxOptionsRole  string
	SELinuxOptionsType  string
	SELinuxOptionsLevel string
}

// WindowsOptionsFields contains Windows options as a flat structure.
type WindowsOptionsFields struct {
	WindowsOptionsGMSACredentialSpecName *string
	WindowsOptionsGMSACredentialSpec     *string
	WindowsOptionsRunAsUserName          *string
}

// SeccompProfileFields contains Seccomp profile options as a flat structure.
type SeccompProfileFields struct {
	SeccompProfileType             v1.SeccompProfileType
	SeccompProfileLocalhostProfile *string
}

// CommonSecurityContextFields contains all security options common to a pod and container.
type CommonSecurityContextFields struct {
	SELinuxOptionsFields
	WindowsOptionsFields
	SeccompProfileFields
	RunAsUser    *int64
	RunAsGroup   *int64
	RunAsNonRoot *bool
}

// PodSecurityContextFields contains all security options specific to a pod.
type PodSecurityContextFields struct {
	CommonSecurityContextFields
	SupplementalGroups  []int64
	FSGroup             *int64
	Sysctls             []v1.Sysctl
	FSGroupChangePolicy *v1.PodFSGroupChangePolicy
}

// SecurityContextFields contains all securoty options specific to a container.
type SecurityContextFields struct {
	CommonSecurityContextFields
	CapabilitiesAdd          []v1.Capability
	CapabilitiesDrop         []v1.Capability
	Privileged               *bool
	ReadOnlyRootFilesystem   *bool
	AllowPrivilegeEscalation *bool
	ProcMount                *v1.ProcMountType
}

// AffinityFields struct holds flat affinity fields.
type AffinityFields struct {
	NodeAffinity    *v1.NodeAffinity
	PodAffinity     *v1.PodAffinity
	PodAntiAffinity *v1.PodAntiAffinity
}

// DNSConfigFields struct holds DNS configuration fields.
type DNSConfigFields struct {
	DNSConfigNameservers []string
	DNSConfigSearches    []string
	DNSConfigOptions     []v1.PodDNSConfigOption
}

// CommonPodFields contains relevant fields from pod specification.
// This flattens some of the embedded structures like security context, DNS config etc.
type CommonPodFields struct {
	PodSecurityContextFields
	AffinityFields
	DNSConfigFields

	NodeSelector                  map[string]string
	RestartPolicy                 v1.RestartPolicy
	TerminationGracePeriodSeconds *int64
	ActiveDeadlineSeconds         *int64
	DNSPolicy                     v1.DNSPolicy
	ServiceAccountName            string
	AutomountServiceAccountToken  *bool
	NodeName                      string
	HostNetwork                   bool
	HostPID                       bool
	HostIPC                       bool
	ShareProcessNamespace         *bool
	ImagePullSecrets              []v1.LocalObjectReference
	Hostname                      string
	Subdomain                     string
	SchedulerName                 string
	Tolerations                   []v1.Toleration
	HostAliases                   []v1.HostAlias
	PriorityClassName             string
	Priority                      *int32
	ReadinessGates                []v1.PodReadinessGate
	RuntimeClassName              *string
	EnableServiceLinks            *bool
	PreemptionPolicy              *v1.PreemptionPolicy
	Overhead                      v1.ResourceList
	TopologySpreadConstraints     []v1.TopologySpreadConstraint
	SetHostnameAsFQDN             *bool
}

// GetCommonPodFields converts pod specification to CommonPodFields structure.
// This flattens some of the embedded structures like security context, DNS config etc.
func GetCommonPodFields(p v1.PodSpec) CommonPodFields {
	item := CommonPodFields{
		NodeSelector:                  p.NodeSelector,
		RestartPolicy:                 p.RestartPolicy,
		TerminationGracePeriodSeconds: p.TerminationGracePeriodSeconds,
		ActiveDeadlineSeconds:         p.ActiveDeadlineSeconds,
		DNSPolicy:                     p.DNSPolicy,
		ServiceAccountName:            p.ServiceAccountName,
		AutomountServiceAccountToken:  p.AutomountServiceAccountToken,
		NodeName:                      p.NodeName,
		HostNetwork:                   p.HostNetwork,
		HostPID:                       p.HostPID,
		HostIPC:                       p.HostIPC,
		ShareProcessNamespace:         p.ShareProcessNamespace,
		ImagePullSecrets:              p.ImagePullSecrets,
		Hostname:                      p.Hostname,
		Subdomain:                     p.Subdomain,
		SchedulerName:                 p.SchedulerName,
		Tolerations:                   p.Tolerations,
		HostAliases:                   p.HostAliases,
		PriorityClassName:             p.PriorityClassName,
		Priority:                      p.Priority,
		ReadinessGates:                p.ReadinessGates,
		RuntimeClassName:              p.RuntimeClassName,
		EnableServiceLinks:            p.EnableServiceLinks,
		PreemptionPolicy:              p.PreemptionPolicy,
		Overhead:                      p.Overhead,
		TopologySpreadConstraints:     p.TopologySpreadConstraints,
		SetHostnameAsFQDN:             p.SetHostnameAsFQDN,
	}
	if p.Affinity != nil {
		item.NodeAffinity = p.Affinity.NodeAffinity
		item.PodAffinity = p.Affinity.PodAffinity
		item.PodAntiAffinity = p.Affinity.PodAntiAffinity
	}
	if p.DNSConfig != nil {
		item.DNSConfigNameservers = p.DNSConfig.Nameservers
		item.DNSConfigSearches = p.DNSConfig.Searches
		item.DNSConfigOptions = p.DNSConfig.Options
	}
	if p.SecurityContext != nil {
		item.RunAsUser = p.SecurityContext.RunAsUser
		item.RunAsGroup = p.SecurityContext.RunAsGroup
		item.RunAsNonRoot = p.SecurityContext.RunAsNonRoot
		item.SupplementalGroups = p.SecurityContext.SupplementalGroups
		item.Sysctls = p.SecurityContext.Sysctls
		item.FSGroup = p.SecurityContext.FSGroup
		item.FSGroupChangePolicy = p.SecurityContext.FSGroupChangePolicy
		if p.SecurityContext.SeccompProfile != nil {
			item.SeccompProfileType = p.SecurityContext.SeccompProfile.Type
			item.SeccompProfileLocalhostProfile = p.SecurityContext.SeccompProfile.LocalhostProfile
		}
		if p.SecurityContext.SELinuxOptions != nil {
			item.SELinuxOptionsLevel = p.SecurityContext.SELinuxOptions.Level
			item.SELinuxOptionsRole = p.SecurityContext.SELinuxOptions.Role
			item.SELinuxOptionsType = p.SecurityContext.SELinuxOptions.Type
			item.SELinuxOptionsUser = p.SecurityContext.SELinuxOptions.User
		}
		if p.SecurityContext.WindowsOptions != nil {
			item.WindowsOptionsRunAsUserName = p.SecurityContext.WindowsOptions.RunAsUserName
			item.WindowsOptionsGMSACredentialSpec = p.SecurityContext.WindowsOptions.GMSACredentialSpec
			item.WindowsOptionsGMSACredentialSpecName = p.SecurityContext.WindowsOptions.GMSACredentialSpecName
		}
	}
	return item
}

// CommonContainerFields contains relevant fields from container specification.
// This flattens some of the embedded structures like security context.
type CommonContainerFields struct {
	SecurityContextFields
	TargetContainerName      string
	Image                    string
	Command                  []string
	Args                     []string
	WorkingDir               string
	Ports                    []v1.ContainerPort
	EnvFrom                  []v1.EnvFromSource
	Env                      []v1.EnvVar
	ResourceLimits           v1.ResourceList
	ResourceRequests         v1.ResourceList
	VolumeMounts             []v1.VolumeMount
	VolumeDevices            []v1.VolumeDevice
	LivenessProbe            *v1.Probe
	ReadinessProbe           *v1.Probe
	StartupProbe             *v1.Probe
	Lifecycle                *v1.Lifecycle
	TerminationMessagePath   string
	TerminationMessagePolicy v1.TerminationMessagePolicy
	ImagePullPolicy          v1.PullPolicy
	Stdin                    bool
	StdinOnce                bool
	TTY                      bool
}

// GetCommonContainerFields converts container specification to CommonContainerFields structure.
// This flattens some of the embedded structures like security context.
func GetCommonContainerFields(c v1.Container) CommonContainerFields {
	item := CommonContainerFields{
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
	}
	copyContainerSecurityContext(&item, c.SecurityContext)
	return item
}

// GetCommonEphemeralContainerFields converts ephemeral container specification to CommonContainerFields.
// This flattens some of the embedded structures like security context.
// Ephemeral container contains one additional field (TargetContainerName) on top of container.
func GetCommonEphemeralContainerFields(c v1.EphemeralContainer) CommonContainerFields {
	item := CommonContainerFields{
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
	}
	copyContainerSecurityContext(&item, c.SecurityContext)
	return item
}

func copyContainerSecurityContext(item *CommonContainerFields, sc *v1.SecurityContext) {
	if sc != nil {
		item.Privileged = sc.Privileged
		item.RunAsUser = sc.RunAsUser
		item.RunAsGroup = sc.RunAsGroup
		item.RunAsNonRoot = sc.RunAsNonRoot
		item.ReadOnlyRootFilesystem = sc.ReadOnlyRootFilesystem
		item.AllowPrivilegeEscalation = sc.AllowPrivilegeEscalation
		item.ProcMount = sc.ProcMount

		if sc.Capabilities != nil {
			item.CapabilitiesAdd = sc.Capabilities.Add
			item.CapabilitiesDrop = sc.Capabilities.Drop
		}
		if sc.SeccompProfile != nil {
			item.SeccompProfileType = sc.SeccompProfile.Type
			item.SeccompProfileLocalhostProfile = sc.SeccompProfile.LocalhostProfile
		}
		if sc.SELinuxOptions != nil {
			item.SELinuxOptionsLevel = sc.SELinuxOptions.Level
			item.SELinuxOptionsRole = sc.SELinuxOptions.Role
			item.SELinuxOptionsType = sc.SELinuxOptions.Type
			item.SELinuxOptionsUser = sc.SELinuxOptions.User
		}
		if sc.WindowsOptions != nil {
			item.WindowsOptionsRunAsUserName = sc.WindowsOptions.RunAsUserName
			item.WindowsOptionsGMSACredentialSpec = sc.WindowsOptions.GMSACredentialSpec
			item.WindowsOptionsGMSACredentialSpecName = sc.WindowsOptions.GMSACredentialSpecName
		}
	}
}

// CommonVolumeFields contains flattened fields from volume specification.
type CommonVolumeFields struct {
	VolumeType                     string
	FSType                         *string
	ReadOnly                       *bool
	SecretName                     string
	HostPathPath                   string
	HostPathType                   *v1.HostPathType
	EmptyDirMedium                 v1.StorageMedium
	EmptyDirSizeLimit              string
	GCEPersistentDiskPDName        string
	GCEPersistentDiskPartition     int32
	AWSElasticBlockStoreVolumeID   string
	AWSElasticBlockStorePartition  int32
	GitRepoRepository              string
	GitRepoRevision                string
	GitRepoDirectory               string
	SecretItems                    []v1.KeyToPath
	SecretDefaultMode              *int32
	SecretOptional                 *bool
	NFSServer                      string
	NFSPath                        string
	ISCSITargetPortal              string
	ISCSIIqn                       string
	ISCSILun                       int32
	ISCSIInterface                 string
	ISCSIPortals                   []string
	ISCSIDiscoveryCHAPAuth         bool
	ISCSISessionCHAPAuth           bool
	ISCSIInitiatorName             *string
	GlusterfsEndpointsName         string
	GlusterfsPath                  string
	PersistentVolumeClaimName      string
	RBDCephMonitors                []string
	RBDImage                       string
	RBDPool                        string
	RBDRadosUser                   string
	RBDKeyring                     string
	FlexVolumeDriver               string
	FlexVolumeOptions              map[string]string
	CinderVolumeID                 string
	CephFSMonitors                 []string
	CephFSPath                     string
	CephFSUser                     string
	CephFSSecretFile               string
	FlockerDatasetName             string
	FlockerDatasetUUID             string
	DownwardAPIItems               []v1.DownwardAPIVolumeFile
	DownwardAPIDefaultMode         *int32
	FCTargetWWNs                   []string
	FCLun                          *int32
	FcWWIDs                        []string
	AzureFileShareName             string
	ConfigMapName                  string
	ConfigMapItems                 []v1.KeyToPath
	ConfigMapDefaultMode           *int32
	ConfigMapOptional              *bool
	VsphereVolumeVolumePath        string
	VsphereVolumeStoragePolicyName string
	VsphereVolumeStoragePolicyID   string
	QuobyteRegistry                string
	QuobyteVolume                  string
	QuobyteUser                    string
	QuobyteGroup                   string
	QuobyteTenant                  string
	AzureDiskDiskName              string
	AzureDiskDataDiskURI           string
	AzureDiskCachingMode           *v1.AzureDataDiskCachingMode
	AzureDiskKind                  *v1.AzureDataDiskKind
	PhotonPersistentDiskPdID       string
	ProjectedSources               []v1.VolumeProjection
	ProjectedDefaultMode           *int32
	PortworxVolumeID               string
	ScaleIOGateway                 string
	ScaleIOSystem                  string
	ScaleIOSSLEnabled              bool
	ScaleIOProtectionDomain        string
	ScaleIOStoragePool             string
	ScaleIOStorageMode             string
	ScaleIOVolumeName              string
	StorageOSVolumeName            string
	StorageOSVolumeNamespace       string
	CSIDriver                      string
	CSIVolumeAttributes            map[string]string
	EphemeralVolumeClaimTemplate   *v1.PersistentVolumeClaimTemplate
}

// GetCommonVolumeFields converts volume specification to CommonVolumeFields.
// This flattens most of the embedded structures like AWSElasticBlockStore, AzureDisk, etc.
func GetCommonVolumeFields(from v1.Volume) CommonVolumeFields {
	to := CommonVolumeFields{}
	if from.AWSElasticBlockStore != nil {
		to.VolumeType = "aws_elastic_block_store"
		to.AWSElasticBlockStoreVolumeID = from.AWSElasticBlockStore.VolumeID
		to.AWSElasticBlockStorePartition = from.AWSElasticBlockStore.Partition
		to.FSType = &from.AWSElasticBlockStore.FSType
		to.ReadOnly = &from.AWSElasticBlockStore.ReadOnly
	}
	if from.AzureDisk != nil {
		to.VolumeType = "azure_disk"
		to.AzureDiskCachingMode = from.AzureDisk.CachingMode
		to.AzureDiskDataDiskURI = from.AzureDisk.DataDiskURI
		to.AzureDiskDiskName = from.AzureDisk.DiskName
		to.AzureDiskKind = from.AzureDisk.Kind
		to.FSType = from.AzureDisk.FSType
		to.ReadOnly = from.AzureDisk.ReadOnly
	}
	if from.AzureFile != nil {
		to.VolumeType = "azure_file"
		to.AzureFileShareName = from.AzureFile.ShareName
		to.SecretName = from.AzureFile.SecretName
		to.ReadOnly = &from.AzureFile.ReadOnly
	}
	if from.CSI != nil {
		to.VolumeType = "csi"
		to.CSIDriver = from.CSI.Driver
		to.CSIVolumeAttributes = from.CSI.VolumeAttributes
		to.FSType = from.CSI.FSType
		to.ReadOnly = from.CSI.ReadOnly
		if from.CSI.NodePublishSecretRef != nil {
			to.SecretName = from.CSI.NodePublishSecretRef.Name
		}
	}
	if from.CephFS != nil {
		to.VolumeType = "ceph_fs"
		to.CephFSMonitors = from.CephFS.Monitors
		to.CephFSPath = from.CephFS.Path
		to.CephFSSecretFile = from.CephFS.SecretFile
		to.CephFSUser = from.CephFS.User
		to.ReadOnly = &from.CephFS.ReadOnly
		if from.CephFS.SecretRef != nil {
			to.SecretName = from.CephFS.SecretRef.Name
		}
	}
	if from.Cinder != nil {
		to.VolumeType = "cinder"
		to.CinderVolumeID = from.Cinder.VolumeID
		to.FSType = &from.Cinder.FSType
		to.ReadOnly = &from.Cinder.ReadOnly
		if from.Cinder.SecretRef != nil {
			to.SecretName = from.Cinder.SecretRef.Name
		}
	}
	if from.ConfigMap != nil {
		to.VolumeType = "config_map"
		to.ConfigMapDefaultMode = from.ConfigMap.DefaultMode
		to.ConfigMapItems = from.ConfigMap.Items
		to.ConfigMapName = from.ConfigMap.Name
		to.ConfigMapOptional = from.ConfigMap.Optional
	}
	if from.DownwardAPI != nil {
		to.VolumeType = "downward_api"
		to.DownwardAPIDefaultMode = from.DownwardAPI.DefaultMode
		to.DownwardAPIItems = from.DownwardAPI.Items
	}
	if from.EmptyDir != nil {
		to.VolumeType = "empty_dir"
		to.EmptyDirMedium = from.EmptyDir.Medium
		to.EmptyDirSizeLimit = from.EmptyDir.SizeLimit.String()
	}
	if from.Ephemeral != nil {
		to.VolumeType = "ephemeral"
		to.EphemeralVolumeClaimTemplate = from.Ephemeral.VolumeClaimTemplate
	}
	if from.FC != nil {
		to.VolumeType = "fc"
		to.FCLun = from.FC.Lun
		to.FCTargetWWNs = from.FC.TargetWWNs
		to.FcWWIDs = from.FC.WWIDs
		to.FSType = &from.FC.FSType
		to.ReadOnly = &from.FC.ReadOnly
	}
	if from.FlexVolume != nil {
		to.VolumeType = "flex_volume"
		to.FlexVolumeDriver = from.FlexVolume.Driver
		to.FlexVolumeOptions = from.FlexVolume.Options
		to.FSType = &from.FlexVolume.FSType
		to.ReadOnly = &from.FlexVolume.ReadOnly
		if from.FlexVolume.SecretRef != nil {
			to.SecretName = from.FlexVolume.SecretRef.Name
		}
	}
	if from.Flocker != nil {
		to.VolumeType = "flocker"
		to.FlockerDatasetName = from.Flocker.DatasetName
		to.FlockerDatasetUUID = from.Flocker.DatasetUUID
	}
	if from.GCEPersistentDisk != nil {
		to.VolumeType = "gce_persistent_disk"
		to.GCEPersistentDiskPDName = from.GCEPersistentDisk.PDName
		to.GCEPersistentDiskPartition = from.GCEPersistentDisk.Partition
		to.FSType = &from.GCEPersistentDisk.FSType
		to.ReadOnly = &from.GCEPersistentDisk.ReadOnly
	}
	if from.GitRepo != nil {
		to.VolumeType = "git_repo"
		to.GitRepoDirectory = from.GitRepo.Directory
		to.GitRepoRepository = from.GitRepo.Repository
		to.GitRepoRevision = from.GitRepo.Revision
	}
	if from.Glusterfs != nil {
		to.VolumeType = "gluster_fs"
		to.GlusterfsPath = from.Glusterfs.Path
		to.GlusterfsEndpointsName = from.Glusterfs.EndpointsName
		to.ReadOnly = &from.Glusterfs.ReadOnly
	}
	if from.HostPath != nil {
		to.VolumeType = "host_path"
		to.HostPathPath = from.HostPath.Path
		to.HostPathType = from.HostPath.Type
	}
	if from.ISCSI != nil {
		to.VolumeType = "iscsci"
		to.ISCSITargetPortal = from.ISCSI.TargetPortal
		to.ISCSIIqn = from.ISCSI.IQN
		to.ISCSILun = from.ISCSI.Lun
		to.ISCSIInterface = from.ISCSI.ISCSIInterface
		to.ISCSIPortals = from.ISCSI.Portals
		to.ISCSIDiscoveryCHAPAuth = from.ISCSI.DiscoveryCHAPAuth
		to.ISCSISessionCHAPAuth = from.ISCSI.SessionCHAPAuth
		to.ISCSIInitiatorName = from.ISCSI.InitiatorName
		to.FSType = &from.ISCSI.FSType
		to.ReadOnly = &from.ISCSI.ReadOnly
		if from.ISCSI.SecretRef != nil {
			to.SecretName = from.ISCSI.SecretRef.Name
		}
	}
	if from.NFS != nil {
		to.VolumeType = "nfs"
		to.NFSPath = from.NFS.Path
		to.NFSServer = from.NFS.Server
		to.ReadOnly = &from.NFS.ReadOnly
	}
	if from.PersistentVolumeClaim != nil {
		to.VolumeType = "persistent_volume_claim"
		to.PersistentVolumeClaimName = from.PersistentVolumeClaim.ClaimName
		to.ReadOnly = &from.PersistentVolumeClaim.ReadOnly
	}
	if from.PhotonPersistentDisk != nil {
		to.VolumeType = "photon_persistent_disk"
		to.PhotonPersistentDiskPdID = from.PhotonPersistentDisk.PdID
		to.FSType = &from.PhotonPersistentDisk.FSType
	}
	if from.PortworxVolume != nil {
		to.VolumeType = "portworx_volume"
		to.PortworxVolumeID = from.PortworxVolume.VolumeID
		to.FSType = &from.PortworxVolume.FSType
		to.ReadOnly = &from.PortworxVolume.ReadOnly
	}
	if from.Projected != nil {
		to.VolumeType = "projected"
		to.ProjectedDefaultMode = from.Projected.DefaultMode
		to.ProjectedSources = from.Projected.Sources
	}
	if from.Quobyte != nil {
		to.VolumeType = "quobyte"
		to.QuobyteGroup = from.Quobyte.Group
		to.QuobyteRegistry = from.Quobyte.Registry
		to.QuobyteTenant = from.Quobyte.Tenant
		to.QuobyteUser = from.Quobyte.User
		to.QuobyteVolume = from.Quobyte.Volume
		to.ReadOnly = &from.Quobyte.ReadOnly
	}
	if from.RBD != nil {
		to.VolumeType = "rbd"
		to.RBDCephMonitors = from.RBD.CephMonitors
		to.RBDImage = from.RBD.RBDImage
		to.RBDPool = from.RBD.RBDPool
		to.RBDRadosUser = from.RBD.RadosUser
		to.RBDKeyring = from.RBD.Keyring
		to.FSType = &from.RBD.FSType
		to.ReadOnly = &from.RBD.ReadOnly
		if from.RBD.SecretRef != nil {
			to.SecretName = from.RBD.SecretRef.Name
		}
	}
	if from.ScaleIO != nil {
		to.VolumeType = "scaleio"
		to.ScaleIOGateway = from.ScaleIO.Gateway
		to.ScaleIOSystem = from.ScaleIO.System
		to.ScaleIOSSLEnabled = from.ScaleIO.SSLEnabled
		to.ScaleIOProtectionDomain = from.ScaleIO.ProtectionDomain
		to.ScaleIOStoragePool = from.ScaleIO.StoragePool
		to.ScaleIOStorageMode = from.ScaleIO.StorageMode
		to.ScaleIOVolumeName = from.ScaleIO.VolumeName
		to.FSType = &from.ScaleIO.FSType
		to.ReadOnly = &from.ScaleIO.ReadOnly
		if from.ScaleIO.SecretRef != nil {
			to.SecretName = from.ScaleIO.SecretRef.Name
		}
	}
	if from.Secret != nil {
		to.VolumeType = "secret"
		to.SecretName = from.Secret.SecretName
		to.SecretItems = from.Secret.Items
		to.SecretDefaultMode = from.Secret.DefaultMode
		to.SecretOptional = from.Secret.Optional
	}
	if from.StorageOS != nil {
		to.VolumeType = "storage_os"
		to.StorageOSVolumeName = from.StorageOS.VolumeName
		to.StorageOSVolumeNamespace = from.StorageOS.VolumeNamespace
		to.FSType = &from.StorageOS.FSType
		to.ReadOnly = &from.StorageOS.ReadOnly
		if from.StorageOS.SecretRef != nil {
			to.SecretName = from.StorageOS.SecretRef.Name
		}
	}
	if from.VsphereVolume != nil {
		to.VolumeType = "vsphere_volume"
		to.VsphereVolumeStoragePolicyID = from.VsphereVolume.StoragePolicyID
		to.VsphereVolumeStoragePolicyName = from.VsphereVolume.StoragePolicyName
		to.VsphereVolumeVolumePath = from.VsphereVolume.VolumePath
		to.FSType = &from.VsphereVolume.FSType
	}
	return to
}
