package apfs

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"howett.net/plist"
)

// diskutilExecTimeout bounds each diskutil invocation so a stuck volume cannot
// block an osquery generate call indefinitely.
const diskutilExecTimeout = 30 * time.Second

// runDiskutil runs diskutil with the given arguments, honoring the query
// context and applying diskutilExecTimeout.
func runDiskutil(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, diskutilExecTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "/usr/sbin/diskutil", args...).Output()
}

// unmarshalCmdResult parses `diskutil apfs list -plist` output.
func unmarshalCmdResult(out []byte) (CmdResult, error) {
	var m CmdResult
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return CmdResult{}, fmt.Errorf("parse diskutil apfs list -plist output: %w", err)
	}
	return m, nil
}

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
	m, err := unmarshalCmdResult(out)
	if err != nil {
		return nil, err
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
	m, err := unmarshalCmdResult(out)
	if err != nil {
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

type cryptoUsersResult struct {
	Users []cryptoUser
}

type cryptoUser struct {
	APFSCryptoUserType string
	APFSCryptoUserUUID string
	VolumeOwner        bool
}

// CryptoUsersColumns is the schema of the apfs_crypto_users table.
func CryptoUsersColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("device_identifier"),
		table.TextColumn("volume_uuid"),
		table.TextColumn("crypto_user_uuid"),
		table.TextColumn("type"),
		table.IntegerColumn("volume_owner"),
	}
}

// CryptoUsersGenerate returns the APFS cryptographic users for each APFS
// container. Cryptographic users include local user accounts with a Secure
// Token, the personal recovery key, and the MDM bootstrap token; the
// volume_owner column reports whether each is an APFS volume owner. This is the
// signal admins use to find hosts where no local user owns the volume (and thus
// cannot authorize OS updates).
func CryptoUsersGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	out, err := runDiskutil(ctx, "apfs", "list", "-plist")
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	return generateCryptoUsers(out, func(device string) ([]byte, error) {
		return runDiskutil(ctx, "apfs", "listCryptoUsers", "-plist", device)
	})
}

// generateCryptoUsers builds apfs_crypto_users rows from `diskutil apfs list
// -plist` output, using listCryptoUsers to fetch the cryptographic users for a
// given volume device. It is separated from CryptoUsersGenerate so the
// orchestration can be unit tested without shelling out to diskutil.
func generateCryptoUsers(listOut []byte, listCryptoUsers func(device string) ([]byte, error)) ([]map[string]string, error) {
	result, err := unmarshalCmdResult(listOut)
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]string, 0)
	for _, container := range result.Containers {
		// All volumes in an APFS volume group report the same cryptographic
		// users, so querying every volume would emit duplicates. Dedup by
		// cryptographic user UUID within the container to drop any overlap.
		seen := make(map[string]struct{})
		for _, volume := range volumesToQuery(container) {
			cryptoOut, err := listCryptoUsers(volume.DeviceIdentifier)
			if err != nil {
				// A volume may not support cryptographic users; skip it rather
				// than failing the whole table.
				log.Debug().Err(err).Str("device", volume.DeviceIdentifier).Msg("failed to list APFS cryptographic users")
				continue
			}

			users, err := parseCryptoUsers(cryptoOut)
			if err != nil {
				log.Debug().Err(err).Str("device", volume.DeviceIdentifier).Msg("failed to parse APFS cryptographic users")
				continue
			}

			for _, user := range users {
				if _, ok := seen[user.APFSCryptoUserUUID]; ok {
					continue
				}
				seen[user.APFSCryptoUserUUID] = struct{}{}
				rows = append(rows, map[string]string{
					"device_identifier": volume.DeviceIdentifier,
					"volume_uuid":       volume.APFSVolumeUUID,
					"crypto_user_uuid":  user.APFSCryptoUserUUID,
					"type":              user.APFSCryptoUserType,
					"volume_owner":      convertBool(user.VolumeOwner),
				})
			}
		}
	}

	return rows, nil
}

// volumesToQuery returns the volumes whose cryptographic users should be listed
// for a container: the Data-role volumes if any exist, otherwise every volume
// in the container. Preferring the Data role keeps this to a single diskutil
// call per container in the common case, while the fallback ensures we don't
// silently return nothing on layouts where the boot volume's role isn't
// reported as "Data".
func volumesToQuery(container Container) []Volume {
	dataVolumes := make([]Volume, 0)
	for _, volume := range container.Volumes {
		for _, role := range volume.Roles {
			if role == "Data" {
				dataVolumes = append(dataVolumes, volume)
				break
			}
		}
	}
	if len(dataVolumes) > 0 {
		return dataVolumes
	}
	return container.Volumes
}

// parseCryptoUsers parses `diskutil apfs listCryptoUsers -plist <device>`
// output into its cryptographic users.
func parseCryptoUsers(out []byte) ([]cryptoUser, error) {
	var result cryptoUsersResult
	if _, err := plist.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse diskutil apfs listCryptoUsers -plist output: %w", err)
	}
	return result.Users, nil
}

func convertBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
