//go:build darwin
// +build darwin

package apfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDiskutilVolumes(t *testing.T) {
	const sampleOutput = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Containers</key>
	<array>
		<dict>
			<key>APFSContainerUUID</key>
			<string>57770A0F-0637-44E3-94BE-52D37EAFB88E</string>
			<key>CapacityCeiling</key>
			<integer>524288000</integer>
			<key>CapacityFree</key>
			<integer>505962496</integer>
			<key>ContainerReference</key>
			<string>disk1</string>
			<key>DesignatedPhysicalStore</key>
			<string>disk0s1</string>
			<key>Fusion</key>
			<false/>
			<key>PhysicalStores</key>
			<array>
			</array>
			<key>Volumes</key>
			<array>
				<dict>
					<key>APFSVolumeUUID</key>
					<string>10DC02F1-71D9-4D8C-A666-899BFDFE2058</string>
					<key>CapacityInUse</key>
					<integer>6475776</integer>
					<key>CapacityQuota</key>
					<integer>0</integer>
					<key>CapacityReserve</key>
					<integer>0</integer>
					<key>CryptoMigrationOn</key>
					<false/>
					<key>DeviceIdentifier</key>
					<string>disk1s1</string>
					<key>Encryption</key>
					<false/>
					<key>FileVault</key>
					<false/>
					<key>Locked</key>
					<false/>
					<key>Name</key>
					<string>iSCPreboot</string>
					<key>Roles</key>
					<array>
						<string>Preboot</string>
					</array>
				</dict>
				<dict>
					<key>APFSVolumeUUID</key>
					<string>AD45A111-EF76-4A09-9D8D-CFB0162952F8</string>
					<key>CapacityInUse</key>
					<integer>6311936</integer>
					<key>CapacityQuota</key>
					<integer>0</integer>
					<key>CapacityReserve</key>
					<integer>0</integer>
					<key>CryptoMigrationOn</key>
					<false/>
					<key>DeviceIdentifier</key>
					<string>disk1s2</string>
					<key>Encryption</key>
					<true/>
					<key>FileVault</key>
					<true/>
					<key>Locked</key>
					<false/>
					<key>Name</key>
					<string>xART</string>
					<key>Roles</key>
					<array>
						<string>xART</string>
					</array>
				</dict>
			</array>
		</dict>
	</array>
</dict>`
	parseResult, err := parseDiskutilVolumes([]byte(sampleOutput))
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{
			"container_uuid":                      "57770A0F-0637-44E3-94BE-52D37EAFB88E",
			"container_designated_physical_store": "disk0s1",
			"container_reference":                 "disk1",
			"container_fusion":                    "0",
			"container_capacity_ceiling":          "524288000",
			"container_capacity_free":             "505962496",
			"uuid":                                "10DC02F1-71D9-4D8C-A666-899BFDFE2058",
			"device_identifier":                   "disk1s1",
			"name":                                "iSCPreboot",
			"role":                                "Preboot",
			"capacity_in_use":                     "6475776",
			"capacity_quota":                      "0",
			"capacity_reserve":                    "0",
			"crypto_migration_on":                 "0",
			"encryption":                          "0",
			"filevault":                           "0",
			"locked":                              "0",
		},
		{
			"container_uuid":                      "57770A0F-0637-44E3-94BE-52D37EAFB88E",
			"container_designated_physical_store": "disk0s1",
			"container_reference":                 "disk1",
			"container_fusion":                    "0",
			"container_capacity_ceiling":          "524288000",
			"container_capacity_free":             "505962496",
			"uuid":                                "AD45A111-EF76-4A09-9D8D-CFB0162952F8",
			"device_identifier":                   "disk1s2",
			"name":                                "xART",
			"role":                                "xART",
			"capacity_in_use":                     "6311936",
			"capacity_quota":                      "0",
			"capacity_reserve":                    "0",
			"crypto_migration_on":                 "0",
			"encryption":                          "1",
			"filevault":                           "1",
			"locked":                              "0",
		},
	}, parseResult)
}

func TestParseDiskutilPhysicalStores(t *testing.T) {
	const sampleOutput = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Containers</key>
	<array>
		<dict>
			<key>APFSContainerUUID</key>
			<string>57770A0F-0637-44E3-94BE-52D37EAFB88E</string>
			<key>CapacityCeiling</key>
			<integer>524288000</integer>
			<key>CapacityFree</key>
			<integer>505962496</integer>
			<key>ContainerReference</key>
			<string>disk1</string>
			<key>DesignatedPhysicalStore</key>
			<string>disk0s1</string>
			<key>Fusion</key>
			<false/>
			<key>PhysicalStores</key>
			<array>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>DiskUUID</key>
					<string>01A35F52-1070-47F4-9F11-ACA37BA87A61</string>
					<key>Size</key>
					<integer>524288000</integer>
				</dict>
				<dict>
					<key>DeviceIdentifier</key>
					<string>disk1s2</string>
					<key>DiskUUID</key>
					<string>4483B6EA-22CD-448B-B5AB-5D937CD19CB3</string>
					<key>Size</key>
					<integer>1048576000</integer>
				</dict>
			</array>
			<key>Volumes</key>
			<array>
			</array>
		</dict>
	</array>
</dict>`
	parseResult, err := parseDiskutilPhysicalStores([]byte(sampleOutput))
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{
			"container_uuid":                      "57770A0F-0637-44E3-94BE-52D37EAFB88E",
			"container_designated_physical_store": "disk0s1",
			"container_reference":                 "disk1",
			"container_fusion":                    "0",
			"container_capacity_ceiling":          "524288000",
			"container_capacity_free":             "505962496",
			"uuid":                                "01A35F52-1070-47F4-9F11-ACA37BA87A61",
			"identifier":                          "disk0s1",
			"size":                                "524288000",
		},
		{
			"container_uuid":                      "57770A0F-0637-44E3-94BE-52D37EAFB88E",
			"container_designated_physical_store": "disk0s1",
			"container_reference":                 "disk1",
			"container_fusion":                    "0",
			"container_capacity_ceiling":          "524288000",
			"container_capacity_free":             "505962496",
			"uuid":                                "4483B6EA-22CD-448B-B5AB-5D937CD19CB3",
			"identifier":                          "disk1s2",
			"size":                                "1048576000",
		},
	}, parseResult)
}
