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

type persistentVolume struct {
	k8s.CommonFields

	Capacity                      v1.ResourceList
	AccessModes                   []v1.PersistentVolumeAccessMode
	ClaimRef                      *v1.ObjectReference
	PersistentVolumeReclaimPolicy v1.PersistentVolumeReclaimPolicy
	StorageClassName              string
	MountOptions                  []string
	VolumeMode                    *v1.PersistentVolumeMode
	NodeAffinity                  *v1.VolumeNodeAffinity

	StatusPhase   v1.PersistentVolumePhase
	StatusMessage string
	StatusReason  string

	VolumeType                     string
	FSType                         *string
	ReadOnly                       *bool
	SecretName                     string
	HostPathPath                   string
	HostPathType                   *v1.HostPathType
	GCEPersistentDiskPDName        string
	GCEPersistentDiskPartition     int32
	AWSElasticBlockStoreVolumeID   string
	AWSElasticBlockStorePartition  int32
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
	LocalPath                      string
	GlusterfsEndpointsName         string
	GlusterfsPath                  string
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
	FCTargetWWNs                   []string
	FCLun                          *int32
	FcWWIDs                        []string
	AzureFileShareName             string
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
}

// PersistentVolumeColumns returns kubernetes persistent volume fields as Osquery table columns.
func PersistentVolumeColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&persistentVolume{})
}

// PersistentVolumesGenerate generates the kubernetes persistent volumes as Osquery table data.
func PersistentVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		pvs, err := k8s.GetClient().CoreV1().PersistentVolumes().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, pv := range pvs.Items {
			item := &persistentVolume{
				CommonFields:                  k8s.GetCommonFields(pv.ObjectMeta),
				Capacity:                      pv.Spec.Capacity,
				AccessModes:                   pv.Spec.AccessModes,
				ClaimRef:                      pv.Spec.ClaimRef,
				PersistentVolumeReclaimPolicy: pv.Spec.PersistentVolumeReclaimPolicy,
				StorageClassName:              pv.Spec.StorageClassName,
				MountOptions:                  pv.Spec.MountOptions,
				VolumeMode:                    pv.Spec.VolumeMode,
				NodeAffinity:                  pv.Spec.NodeAffinity,
				StatusPhase:                   pv.Status.Phase,
				StatusMessage:                 pv.Status.Message,
				StatusReason:                  pv.Status.Reason,
			}
			if pv.Spec.AWSElasticBlockStore != nil {
				item.VolumeType = "aws_elastic_block_store"
				item.AWSElasticBlockStoreVolumeID = pv.Spec.AWSElasticBlockStore.VolumeID
				item.AWSElasticBlockStorePartition = pv.Spec.AWSElasticBlockStore.Partition
				item.FSType = &pv.Spec.AWSElasticBlockStore.FSType
				item.ReadOnly = &pv.Spec.AWSElasticBlockStore.ReadOnly
			}
			if pv.Spec.AzureDisk != nil {
				item.VolumeType = "azure_disk"
				item.AzureDiskCachingMode = pv.Spec.AzureDisk.CachingMode
				item.AzureDiskDataDiskURI = pv.Spec.AzureDisk.DataDiskURI
				item.AzureDiskDiskName = pv.Spec.AzureDisk.DiskName
				item.AzureDiskKind = pv.Spec.AzureDisk.Kind
				item.FSType = pv.Spec.AzureDisk.FSType
				item.ReadOnly = pv.Spec.AzureDisk.ReadOnly
			}
			if pv.Spec.AzureFile != nil {
				item.VolumeType = "azure_file"
				item.AzureFileShareName = pv.Spec.AzureFile.ShareName
				item.SecretName = pv.Spec.AzureFile.SecretName
				item.ReadOnly = &pv.Spec.AzureFile.ReadOnly
			}
			if pv.Spec.CSI != nil {
				item.VolumeType = "csi"
				item.CSIDriver = pv.Spec.CSI.Driver
				item.CSIVolumeAttributes = pv.Spec.CSI.VolumeAttributes
				item.FSType = &pv.Spec.CSI.FSType
				item.ReadOnly = &pv.Spec.CSI.ReadOnly
				if pv.Spec.CSI.NodePublishSecretRef != nil {
					item.SecretName = pv.Spec.CSI.NodePublishSecretRef.Name
				}
			}
			if pv.Spec.CephFS != nil {
				item.VolumeType = "ceph_fs"
				item.CephFSMonitors = pv.Spec.CephFS.Monitors
				item.CephFSPath = pv.Spec.CephFS.Path
				item.CephFSSecretFile = pv.Spec.CephFS.SecretFile
				item.CephFSUser = pv.Spec.CephFS.User
				item.ReadOnly = &pv.Spec.CephFS.ReadOnly
				if pv.Spec.CephFS.SecretRef != nil {
					item.SecretName = pv.Spec.CephFS.SecretRef.Name
				}
			}
			if pv.Spec.Cinder != nil {
				item.VolumeType = "cinder"
				item.CinderVolumeID = pv.Spec.Cinder.VolumeID
				item.FSType = &pv.Spec.Cinder.FSType
				item.ReadOnly = &pv.Spec.Cinder.ReadOnly
				if pv.Spec.Cinder.SecretRef != nil {
					item.SecretName = pv.Spec.Cinder.SecretRef.Name
				}
			}
			if pv.Spec.FC != nil {
				item.VolumeType = "fc"
				item.FCLun = pv.Spec.FC.Lun
				item.FCTargetWWNs = pv.Spec.FC.TargetWWNs
				item.FcWWIDs = pv.Spec.FC.WWIDs
				item.FSType = &pv.Spec.FC.FSType
				item.ReadOnly = &pv.Spec.FC.ReadOnly
			}
			if pv.Spec.FlexVolume != nil {
				item.VolumeType = "flex_volume"
				item.FlexVolumeDriver = pv.Spec.FlexVolume.Driver
				item.FlexVolumeOptions = pv.Spec.FlexVolume.Options
				item.FSType = &pv.Spec.FlexVolume.FSType
				item.ReadOnly = &pv.Spec.FlexVolume.ReadOnly
				if pv.Spec.FlexVolume.SecretRef != nil {
					item.SecretName = pv.Spec.FlexVolume.SecretRef.Name
				}
			}
			if pv.Spec.Flocker != nil {
				item.VolumeType = "flocker"
				item.FlockerDatasetName = pv.Spec.Flocker.DatasetName
				item.FlockerDatasetUUID = pv.Spec.Flocker.DatasetUUID
			}
			if pv.Spec.GCEPersistentDisk != nil {
				item.VolumeType = "gce_persistent_disk"
				item.GCEPersistentDiskPDName = pv.Spec.GCEPersistentDisk.PDName
				item.GCEPersistentDiskPartition = pv.Spec.GCEPersistentDisk.Partition
				item.FSType = &pv.Spec.GCEPersistentDisk.FSType
				item.ReadOnly = &pv.Spec.GCEPersistentDisk.ReadOnly
			}
			if pv.Spec.Glusterfs != nil {
				item.VolumeType = "gluster_fs"
				item.GlusterfsPath = pv.Spec.Glusterfs.Path
				item.GlusterfsEndpointsName = pv.Spec.Glusterfs.EndpointsName
				item.ReadOnly = &pv.Spec.Glusterfs.ReadOnly
			}
			if pv.Spec.HostPath != nil {
				item.VolumeType = "host_path"
				item.HostPathPath = pv.Spec.HostPath.Path
				item.HostPathType = pv.Spec.HostPath.Type
			}
			if pv.Spec.ISCSI != nil {
				item.VolumeType = "iscsci"
				item.ISCSITargetPortal = pv.Spec.ISCSI.TargetPortal
				item.ISCSIIqn = pv.Spec.ISCSI.IQN
				item.ISCSILun = pv.Spec.ISCSI.Lun
				item.ISCSIInterface = pv.Spec.ISCSI.ISCSIInterface
				item.ISCSIPortals = pv.Spec.ISCSI.Portals
				item.ISCSIDiscoveryCHAPAuth = pv.Spec.ISCSI.DiscoveryCHAPAuth
				item.ISCSISessionCHAPAuth = pv.Spec.ISCSI.SessionCHAPAuth
				item.ISCSIInitiatorName = pv.Spec.ISCSI.InitiatorName
				item.FSType = &pv.Spec.ISCSI.FSType
				item.ReadOnly = &pv.Spec.ISCSI.ReadOnly
				if pv.Spec.ISCSI.SecretRef != nil {
					item.SecretName = pv.Spec.ISCSI.SecretRef.Name
				}
			}
			if pv.Spec.Local != nil {
				item.LocalPath = pv.Spec.Local.Path
				item.FSType = pv.Spec.Local.FSType
			}
			if pv.Spec.NFS != nil {
				item.VolumeType = "nfs"
				item.NFSPath = pv.Spec.NFS.Path
				item.NFSServer = pv.Spec.NFS.Server
				item.ReadOnly = &pv.Spec.NFS.ReadOnly
			}
			if pv.Spec.PhotonPersistentDisk != nil {
				item.VolumeType = "photon_persistent_disk"
				item.PhotonPersistentDiskPdID = pv.Spec.PhotonPersistentDisk.PdID
				item.FSType = &pv.Spec.PhotonPersistentDisk.FSType
			}
			if pv.Spec.PortworxVolume != nil {
				item.VolumeType = "portworx_volume"
				item.PortworxVolumeID = pv.Spec.PortworxVolume.VolumeID
				item.FSType = &pv.Spec.PortworxVolume.FSType
				item.ReadOnly = &pv.Spec.PortworxVolume.ReadOnly
			}
			if pv.Spec.Quobyte != nil {
				item.VolumeType = "quobyte"
				item.QuobyteGroup = pv.Spec.Quobyte.Group
				item.QuobyteRegistry = pv.Spec.Quobyte.Registry
				item.QuobyteTenant = pv.Spec.Quobyte.Tenant
				item.QuobyteUser = pv.Spec.Quobyte.User
				item.QuobyteVolume = pv.Spec.Quobyte.Volume
				item.ReadOnly = &pv.Spec.Quobyte.ReadOnly
			}
			if pv.Spec.RBD != nil {
				item.VolumeType = "rbd"
				item.RBDCephMonitors = pv.Spec.RBD.CephMonitors
				item.RBDImage = pv.Spec.RBD.RBDImage
				item.RBDPool = pv.Spec.RBD.RBDPool
				item.RBDRadosUser = pv.Spec.RBD.RadosUser
				item.RBDKeyring = pv.Spec.RBD.Keyring
				item.FSType = &pv.Spec.RBD.FSType
				item.ReadOnly = &pv.Spec.RBD.ReadOnly
				if pv.Spec.RBD.SecretRef != nil {
					item.SecretName = pv.Spec.RBD.SecretRef.Name
				}
			}
			if pv.Spec.ScaleIO != nil {
				item.VolumeType = "scaleio"
				item.ScaleIOGateway = pv.Spec.ScaleIO.Gateway
				item.ScaleIOSystem = pv.Spec.ScaleIO.System
				item.ScaleIOSSLEnabled = pv.Spec.ScaleIO.SSLEnabled
				item.ScaleIOProtectionDomain = pv.Spec.ScaleIO.ProtectionDomain
				item.ScaleIOStoragePool = pv.Spec.ScaleIO.StoragePool
				item.ScaleIOStorageMode = pv.Spec.ScaleIO.StorageMode
				item.ScaleIOVolumeName = pv.Spec.ScaleIO.VolumeName
				item.FSType = &pv.Spec.ScaleIO.FSType
				item.ReadOnly = &pv.Spec.ScaleIO.ReadOnly
				if pv.Spec.ScaleIO.SecretRef != nil {
					item.SecretName = pv.Spec.ScaleIO.SecretRef.Name
				}
			}
			if pv.Spec.StorageOS != nil {
				item.VolumeType = "storage_os"
				item.StorageOSVolumeName = pv.Spec.StorageOS.VolumeName
				item.StorageOSVolumeNamespace = pv.Spec.StorageOS.VolumeNamespace
				item.FSType = &pv.Spec.StorageOS.FSType
				item.ReadOnly = &pv.Spec.StorageOS.ReadOnly
				if pv.Spec.StorageOS.SecretRef != nil {
					item.SecretName = pv.Spec.StorageOS.SecretRef.Name
				}
			}
			if pv.Spec.VsphereVolume != nil {
				item.VolumeType = "vsphere_volume"
				item.VsphereVolumeStoragePolicyID = pv.Spec.VsphereVolume.StoragePolicyID
				item.VsphereVolumeStoragePolicyName = pv.Spec.VsphereVolume.StoragePolicyName
				item.VsphereVolumeVolumePath = pv.Spec.VsphereVolume.VolumePath
				item.FSType = &pv.Spec.VsphereVolume.FSType
			}
			results = append(results, k8s.ToMap(item))
		}

		if pvs.Continue == "" {
			break
		}
		options.Continue = pvs.Continue
	}

	return results, nil
}
