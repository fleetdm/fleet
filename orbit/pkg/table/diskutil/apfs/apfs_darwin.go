package apfs

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/osquery/osquery-go/plugin/table"
	"howett.net/plist"
)

type CmdResult struct {
	Containers []Container
}

type Container struct {
	APFSContainerUUID       string
	CapacityCeiling         int64
	CapacityFree            int64
	ContainerReference      string
	DesignatedPhysicalStore string
	Fusion                  bool
	PhysicalStores          []PhysicalStore
	Volumes                 []Volume
}

type PhysicalStore struct {
	DeviceIdentifier string
	DiskUUID         string
	Size             int64
}

type Volume struct {
	APFSVolumeUUID    string
	CapacityInUse     int64
	CapacityQuota     int64
	CapacityReserve   int64
	CryptoMigrationOn bool
	DeviceIdentifier  string
	Encryption        bool
	FileVault         bool
	Locked            bool
	Name              string
	Roles             []string
}

// Columns is the schema of the table.
func VolumesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("container_uuid"),
		table.TextColumn("container_designated_physical_store"),
		table.TextColumn("container_reference"),
		table.IntegerColumn("container_fusion"),
		table.BigIntColumn("container_capacity_ceiling"),
		table.BigIntColumn("container_capacity_free"),
		table.TextColumn("uuid"),
		table.TextColumn("device_identifier"),
		table.TextColumn("name"),
		table.TextColumn("role"),
		table.BigIntColumn("capacity_in_use"),
		table.BigIntColumn("capacity_quota"),
		table.BigIntColumn("capacity_reserve"),
		table.BigIntColumn("crypto_migration_on"),
		table.BigIntColumn("encryption"),
		table.IntegerColumn("filevault"),
		table.IntegerColumn("locked"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func VolumesGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	cmd := exec.Command("/usr/sbin/diskutil", "apfs", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows, err := parseDiskutilVolumes(out)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func parseDiskutilVolumes(out []byte) ([]map[string]string, error) {
	var m CmdResult
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("parse diskutil apfs list -plist output: %w", err)
	}
	rows := make([]map[string]string, 0)

	for _, container := range m.Containers {
		for _, volume := range container.Volumes {
			role := ""
			if len(volume.Roles) > 0 {
				role = volume.Roles[0]
			}
			rows = append(rows, map[string]string{
				"container_uuid":                      container.APFSContainerUUID,
				"container_designated_physical_store": container.DesignatedPhysicalStore,
				"container_reference":                 container.ContainerReference,
				"container_fusion":                    convertBool(container.Fusion),
				"container_capacity_ceiling":          strconv.FormatInt(container.CapacityCeiling, 10),
				"container_capacity_free":             strconv.FormatInt(container.CapacityFree, 10),
				"uuid":                                volume.APFSVolumeUUID,
				"device_identifier":                   volume.DeviceIdentifier,
				"name":                                volume.Name,
				"role":                                role,
				"capacity_in_use":                     strconv.FormatInt(volume.CapacityInUse, 10),
				"capacity_quota":                      strconv.FormatInt(volume.CapacityQuota, 10),
				"capacity_reserve":                    strconv.FormatInt(volume.CapacityReserve, 10),
				"crypto_migration_on":                 convertBool(volume.CryptoMigrationOn),
				"encryption":                          convertBool(volume.Encryption),
				"filevault":                           convertBool(volume.FileVault),
				"locked":                              convertBool(volume.Locked),
			})
		}
	}

	return rows, nil
}

// Columns is the schema of the table.
func PhysicalStoresColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("container_uuid"),
		table.TextColumn("container_designated_physical_store"),
		table.TextColumn("container_reference"),
		table.IntegerColumn("container_fusion"),
		table.BigIntColumn("container_capacity_ceiling"),
		table.BigIntColumn("container_capacity_free"),
		table.TextColumn("uuid"),
		table.TextColumn("identifier"),
		table.BigIntColumn("size"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func PhysicalStoresGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	cmd := exec.Command("/usr/sbin/diskutil", "apfs", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	rows, err := parseDiskutilPhysicalStores(out)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func parseDiskutilPhysicalStores(out []byte) ([]map[string]string, error) {
	var m CmdResult
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0)

	for _, container := range m.Containers {
		for _, physicalStore := range container.PhysicalStores {
			rows = append(rows, map[string]string{
				"container_uuid":                      container.APFSContainerUUID,
				"container_designated_physical_store": container.DesignatedPhysicalStore,
				"container_reference":                 container.ContainerReference,
				"container_fusion":                    convertBool(container.Fusion),
				"container_capacity_ceiling":          strconv.FormatInt(container.CapacityCeiling, 10),
				"container_capacity_free":             strconv.FormatInt(container.CapacityFree, 10),
				"uuid":                                physicalStore.DiskUUID,
				"identifier":                          physicalStore.DeviceIdentifier,
				"size":                                strconv.FormatInt(physicalStore.Size, 10),
			})
		}
	}

	return rows, nil
}

func convertBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
