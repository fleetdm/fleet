package corestorage

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/osquery/osquery-go/plugin/table"
	"howett.net/plist"
)

type ListCmdResult struct {
	CoreStorageLogicalVolumeGroups []LogicalVolumeGroup
}

type LogicalVolumeGroup struct {
	CoreStorageLogicalVolumeFamilies []LogicalVolumeFamily
	CoreStoragePhysicalVolumes       []BasicVolume
	CoreStorageRole                  string
	CoreStorageUUID                  string
}

type LogicalVolumeFamily struct {
	CoreStorageLogicalVolumes []BasicVolume
	CoreStorageRole           string
	CoreStorageUUID           string
}

type BasicVolume struct {
	CoreStorageRole string
	CoreStorageUUID string
}

type VolumeGroupInfo struct {
	CoreStorageLogicalVolumeGroupFreeSpace   uint64
	CoreStorageLogicalVolumeGroupFusionDrive bool
	CoreStorageLogicalVolumeGroupName        string
	CoreStorageLogicalVolumeGroupSequence    uint64
	CoreStorageLogicalVolumeGroupSize        uint64
	CoreStorageLogicalVolumeGroupSparse      bool
	CoreStorageLogicalVolumeGroupStatus      string
	CoreStorageLogicalVolumeGroupVersion     uint64
	CoreStorageRole                          string
	CoreStorageRoleUserVisibleName           string
	CoreStorageUUID                          string
}

type LogicalVolumeFamilyInfo struct {
	CoreStorageLogicalVolumeFamilyEncryptionStatus string
	CoreStorageLogicalVolumeFamilyEncryptionType   string
	CoreStorageQueryContextHasVisibleUsers         bool
	CoreStorageQueryContextHasVolumeKey            bool
	CoreStorageQueryContextIsAcceptingNewUsers     bool
	CoreStorageQueryContextIsFullySecure           bool
	CoreStorageQueryContextMayHaveEncryptedEvents  bool
	CoreStorageQueryContextRequiresPasswordUnlock  bool
	CoreStorageRole                                string
	CoreStorageRoleUserVisibleName                 string
	CoreStorageUUID                                string
	MemberOfCoreStorageLogicalVolumeGroup          string
}

type LogicalVolumeInfo struct {
	CoreStorageLogicalVolumeContentHint                 string
	CoreStorageLogicalVolumeConversionProgressPercent   uint8
	CoreStorageLogicalVolumeConversionState             string
	CoreStorageLogicalVolumeName                        string
	CoreStorageLogicalVolumeSequence                    uint64
	CoreStorageLogicalVolumeSize                        uint64
	CoreStorageLogicalVolumeStatus                      string
	CoreStorageLogicalVolumeVersion                     uint64
	CoreStorageRole                                     string
	CoreStorageRoleUserVisibleName                      string
	CoreStorageUUID                                     string
	DesignatedCoreStoragePhysicalVolume                 string
	DesignatedCoreStoragePhysicalVolumeDeviceIdentifier string
	DeviceIdentifier                                    string
	MemberOfCoreStorageLogicalVolumeFamily              string
	MemberOfCoreStorageLogicalVolumeGroup               string
	VolumeName                                          string
}

type PhysicalVolumeInfo struct {
	CoreStoragePhysicalVolumeIndex        uint64
	CoreStoragePhysicalVolumeSize         uint64
	CoreStoragePhysicalVolumeStatus       string
	CoreStorageRole                       string
	CoreStorageRoleUserVisibleName        string
	CoreStorageUUID                       string
	DeviceIdentifier                      string
	MemberOfCoreStorageLogicalVolumeGroup string
}

// Columns is the schema of the table.
func LogicalVolumesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("vg_UUID"),
		table.IntegerColumn("vg_Version"),
		table.BigIntColumn("vg_FreeSpace"),
		table.IntegerColumn("vg_FusionDrive"),
		table.TextColumn("vg_Name"),
		table.BigIntColumn("vg_Sequence"),
		table.BigIntColumn("vg_Size"),
		table.IntegerColumn("vg_Sparse"),
		table.TextColumn("vg_Status"),
		table.TextColumn("lvf_UUID"),
		table.TextColumn("lvf_EncryptionStatus"),
		table.TextColumn("lvf_EncryptionType"),
		table.IntegerColumn("lvf_HasVisibleUsers"),
		table.IntegerColumn("lvf_HasVolumeKey"),
		table.IntegerColumn("lvf_IsAcceptingNewUsers"),
		table.IntegerColumn("lvf_IsFullySecure"),
		table.IntegerColumn("lvf_MayHaveEncryptedEvents"),
		table.IntegerColumn("lvf_RequiresPasswordUnlock"),
		table.TextColumn("ContentHint"),
		table.IntegerColumn("ConverstionProgressPercent"),
		table.TextColumn("ConversionState"),
		table.TextColumn("Name"),
		table.BigIntColumn("Sequence"),
		table.BigIntColumn("Size"),
		table.TextColumn("Status"),
		table.BigIntColumn("Version"),
		table.TextColumn("UUID"),
		table.TextColumn("DesignatedPhysicalVolume"),
		table.TextColumn("DesignatedPhysicalVolumeIdentifier"),
		table.TextColumn("Identifier"),
		table.TextColumn("VolumeName"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func LogicalVolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	cmd := exec.Command("/usr/sbin/diskutil", "coreStorage", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows, err := parseDiskutilLogicalVolumes(out)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func parseDiskutilLogicalVolumes(out []byte) ([]map[string]string, error) {
	var m ListCmdResult
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("parse diskutil coreStorage list -plist output: %w", err)
	}
	rows := make([]map[string]string, 0)

	for _, vg := range m.CoreStorageLogicalVolumeGroups {
		vg_info, err := runDiskutilInfo[VolumeGroupInfo](vg.CoreStorageUUID)
		if err != nil {
			return nil, fmt.Errorf("get volume group info: %w", err)
		}
		for _, lvf := range vg.CoreStorageLogicalVolumeFamilies {
			lvf_info, err := runDiskutilInfo[LogicalVolumeFamilyInfo](lvf.CoreStorageUUID)
			if err != nil {
				return nil, fmt.Errorf("get logical volume family info: %w", err)
			}
			for _, lv := range lvf.CoreStorageLogicalVolumes {
				lv_info, err := runDiskutilInfo[LogicalVolumeInfo](lv.CoreStorageUUID)
				if err != nil {
					return nil, fmt.Errorf("get logical volume info: %w", err)
				}
				rows = append(rows, map[string]string{
					"vg_UUID":                            vg_info.CoreStorageUUID,
					"vg_Version":                         strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupVersion, 10),
					"vg_FreeSpace":                       strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupFreeSpace, 10),
					"vg_FusionDrive":                     convertBool(vg_info.CoreStorageLogicalVolumeGroupFusionDrive),
					"vg_Name":                            vg_info.CoreStorageLogicalVolumeGroupName,
					"vg_Sequence":                        strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupSequence, 10),
					"vg_Size":                            strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupSize, 10),
					"vg_Sparse":                          convertBool(vg_info.CoreStorageLogicalVolumeGroupSparse),
					"vg_Status":                          vg_info.CoreStorageLogicalVolumeGroupStatus,
					"lvf_UUID":                           lvf_info.CoreStorageUUID,
					"lvf_EncryptionStatus":               lvf_info.CoreStorageLogicalVolumeFamilyEncryptionStatus,
					"lvf_EncryptionType":                 lvf_info.CoreStorageLogicalVolumeFamilyEncryptionType,
					"lvf_HasVisibleUsers":                convertBool(lvf_info.CoreStorageQueryContextHasVisibleUsers),
					"lvf_HasVolumeKey":                   convertBool(lvf_info.CoreStorageQueryContextHasVolumeKey),
					"lvf_IsAcceptingNewUsers":            convertBool(lvf_info.CoreStorageQueryContextIsAcceptingNewUsers),
					"lvf_IsFullySecure":                  convertBool(lvf_info.CoreStorageQueryContextIsFullySecure),
					"lvf_MayHaveEncryptedEvents":         convertBool(lvf_info.CoreStorageQueryContextMayHaveEncryptedEvents),
					"lvf_RequiresPasswordUnlock":         convertBool(lvf_info.CoreStorageQueryContextRequiresPasswordUnlock),
					"ContentHint":                        lv_info.CoreStorageLogicalVolumeContentHint,
					"ConversionProgressPercent":          strconv.FormatUint(uint64(lv_info.CoreStorageLogicalVolumeConversionProgressPercent), 10),
					"ConversionState":                    lv_info.CoreStorageLogicalVolumeConversionState,
					"Name":                               lv_info.CoreStorageLogicalVolumeName,
					"Sequence":                           strconv.FormatUint(lv_info.CoreStorageLogicalVolumeSequence, 10),
					"Size":                               strconv.FormatUint(lv_info.CoreStorageLogicalVolumeSize, 10),
					"Status":                             lv_info.CoreStorageLogicalVolumeStatus,
					"Version":                            strconv.FormatUint(lv_info.CoreStorageLogicalVolumeVersion, 10),
					"UUID":                               lv_info.CoreStorageUUID,
					"DesignatedPhysicalVolume":           lv_info.DesignatedCoreStoragePhysicalVolume,
					"DesignatedPhysicalVolumeIdentifier": lv_info.DesignatedCoreStoragePhysicalVolumeDeviceIdentifier,
					"Identifier":                         lv_info.DeviceIdentifier,
					"VolumeName":                         lv_info.VolumeName,
				})
			}
		}
	}

	return rows, nil
}

func LogicalVolumeFamiliesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("vg_UUID"),
		table.IntegerColumn("vg_Version"),
		table.BigIntColumn("vg_FreeSpace"),
		table.IntegerColumn("vg_FusionDrive"),
		table.TextColumn("vg_Name"),
		table.BigIntColumn("vg_Sequence"),
		table.BigIntColumn("vg_Size"),
		table.IntegerColumn("vg_Sparse"),
		table.TextColumn("vg_Status"),
		table.TextColumn("UUID"),
		table.TextColumn("EncryptionStatus"),
		table.TextColumn("EncryptionType"),
		table.IntegerColumn("HasVisibleUsers"),
		table.IntegerColumn("HasVolumeKey"),
		table.IntegerColumn("IsAcceptingNewUsers"),
		table.IntegerColumn("IsFullySecure"),
		table.IntegerColumn("MayHaveEncryptedEvents"),
		table.IntegerColumn("RequiresPasswordUnlock"),
	}
}

func LogicalVolumeFamiliesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	cmd := exec.Command("/usr/sbin/diskutil", "coreStorage", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows, err := parseDiskutilLogicalVolumeFamilies(out)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func parseDiskutilLogicalVolumeFamilies(out []byte) ([]map[string]string, error) {
	var m ListCmdResult
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("parse diskutil coreStorage list -plist output: %w", err)
	}
	rows := make([]map[string]string, 0)

	for _, vg := range m.CoreStorageLogicalVolumeGroups {
		vg_info, err := runDiskutilInfo[VolumeGroupInfo](vg.CoreStorageUUID)
		if err != nil {
			return nil, fmt.Errorf("get volume group info: %w", err)
		}
		for _, lvf := range vg.CoreStorageLogicalVolumeFamilies {
			lvf_info, err := runDiskutilInfo[LogicalVolumeFamilyInfo](lvf.CoreStorageUUID)
			if err != nil {
				return nil, fmt.Errorf("get logical volume family info: %w", err)
			}
			rows = append(rows, map[string]string{
				"vg_UUID":                vg_info.CoreStorageUUID,
				"vg_Version":             strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupVersion, 10),
				"vg_FreeSpace":           strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupFreeSpace, 10),
				"vg_FusionDrive":         convertBool(vg_info.CoreStorageLogicalVolumeGroupFusionDrive),
				"vg_Name":                vg_info.CoreStorageLogicalVolumeGroupName,
				"vg_Sequence":            strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupSequence, 10),
				"vg_Size":                strconv.FormatUint(vg_info.CoreStorageLogicalVolumeGroupSize, 10),
				"vg_Sparse":              convertBool(vg_info.CoreStorageLogicalVolumeGroupSparse),
				"vg_Status":              vg_info.CoreStorageLogicalVolumeGroupStatus,
				"UUID":                   lvf_info.CoreStorageUUID,
				"EncryptionStatus":       lvf_info.CoreStorageLogicalVolumeFamilyEncryptionStatus,
				"EncryptionType":         lvf_info.CoreStorageLogicalVolumeFamilyEncryptionType,
				"HasVisibleUsers":        convertBool(lvf_info.CoreStorageQueryContextHasVisibleUsers),
				"HasVolumeKey":           convertBool(lvf_info.CoreStorageQueryContextHasVolumeKey),
				"IsAcceptingNewUsers":    convertBool(lvf_info.CoreStorageQueryContextIsAcceptingNewUsers),
				"IsFullySecure":          convertBool(lvf_info.CoreStorageQueryContextIsFullySecure),
				"MayHaveEncryptedEvents": convertBool(lvf_info.CoreStorageQueryContextMayHaveEncryptedEvents),
				"RequiresPasswordUnlock": convertBool(lvf_info.CoreStorageQueryContextRequiresPasswordUnlock),
			})
		}
	}

	return rows, nil
}

func runDiskutilInfo[T interface{}](uuid string) (*T, error) {
	var result T
	cmd := exec.Command("/usr/sbin/diskutil", "coreStorage", "info", "-plist", uuid)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("coreStorage info failed: %w", err)
	}

	_, err = plist.Unmarshal(out, &result)
	if err != nil {
		return nil, fmt.Errorf("coreStorage info parse failed: %w", err)
	}
	return &result, nil
}

func convertBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
