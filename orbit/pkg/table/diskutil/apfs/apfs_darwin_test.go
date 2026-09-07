//go:build darwin
// +build darwin

package apfs

import (
	"fmt"
	"strings"
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

const listCryptoUsersOutput = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Users</key>
	<array>
		<dict>
			<key>APFSCryptoUserType</key>
			<string>LocalOpenDirectory</string>
			<key>APFSCryptoUserUUID</key>
			<string>3C7DB8EE-5EA4-4C18-A7E4-25C6C0F94667</string>
			<key>VolumeOwner</key>
			<true/>
		</dict>
		<dict>
			<key>APFSCryptoUserType</key>
			<string>PersonalRecovery</string>
			<key>APFSCryptoUserUUID</key>
			<string>EBC6C064-0000-11AA-AA11-00306543ECAC</string>
			<key>VolumeOwner</key>
			<true/>
		</dict>
		<dict>
			<key>APFSCryptoUserType</key>
			<string>MDMRecovery</string>
			<key>APFSCryptoUserUUID</key>
			<string>2457711A-523C-4604-B75A-F48A571D5036</string>
			<key>VolumeOwner</key>
			<false/>
		</dict>
	</array>
</dict>`

func TestParseCryptoUsers(t *testing.T) {
	users, err := parseCryptoUsers([]byte(listCryptoUsersOutput))
	require.NoError(t, err)
	require.Equal(t, []cryptoUser{
		{APFSCryptoUserType: "LocalOpenDirectory", APFSCryptoUserUUID: "3C7DB8EE-5EA4-4C18-A7E4-25C6C0F94667", VolumeOwner: true},
		{APFSCryptoUserType: "PersonalRecovery", APFSCryptoUserUUID: "EBC6C064-0000-11AA-AA11-00306543ECAC", VolumeOwner: true},
		{APFSCryptoUserType: "MDMRecovery", APFSCryptoUserUUID: "2457711A-523C-4604-B75A-F48A571D5036", VolumeOwner: false},
	}, users)
}

func TestParseCryptoUsersEmpty(t *testing.T) {
	const emptyOutput = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Users</key>
	<array/>
</dict>`
	users, err := parseCryptoUsers([]byte(emptyOutput))
	require.NoError(t, err)
	require.Empty(t, users)
}

func TestParseCryptoUsersMalformed(t *testing.T) {
	_, err := parseCryptoUsers([]byte("not a plist"))
	require.Error(t, err)
}

func TestVolumesToQuery(t *testing.T) {
	system := Volume{DeviceIdentifier: "disk3s1", Roles: []string{"System"}}
	preboot := Volume{DeviceIdentifier: "disk3s2", Roles: []string{"Preboot"}}
	data := Volume{DeviceIdentifier: "disk3s5", Roles: []string{"Data"}}

	t.Run("prefers data role", func(t *testing.T) {
		got := volumesToQuery(Container{Volumes: []Volume{system, preboot, data}})
		require.Equal(t, []Volume{data}, got)
	})

	t.Run("falls back to all volumes when no data role", func(t *testing.T) {
		got := volumesToQuery(Container{Volumes: []Volume{system, preboot}})
		require.Equal(t, []Volume{system, preboot}, got)
	})
}

// cryptoUsersPlist renders a listCryptoUsers -plist document for the given users.
func cryptoUsersPlist(t *testing.T, users ...cryptoUser) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<plist version=\"1.0\">\n<dict>\n\t<key>Users</key>\n\t<array>\n")
	for _, u := range users {
		owner := "<false/>"
		if u.VolumeOwner {
			owner = "<true/>"
		}
		fmt.Fprintf(&b, "\t\t<dict><key>APFSCryptoUserType</key><string>%s</string><key>APFSCryptoUserUUID</key><string>%s</string><key>VolumeOwner</key>%s</dict>\n", u.APFSCryptoUserType, u.APFSCryptoUserUUID, owner)
	}
	b.WriteString("\t</array>\n</dict>\n</plist>")
	return []byte(b.String())
}

// listPlist renders a `diskutil apfs list -plist` document for a single
// container with the given volumes.
func listPlist(t *testing.T, volumes ...Volume) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<plist version=\"1.0\">\n<dict>\n\t<key>Containers</key>\n\t<array>\n\t\t<dict>\n\t\t\t<key>APFSContainerUUID</key><string>30C51B81-A7B6-4BF5-BB21-E67953FB0EAE</string>\n\t\t\t<key>Volumes</key>\n\t\t\t<array>\n")
	for _, v := range volumes {
		b.WriteString("\t\t\t\t<dict>")
		fmt.Fprintf(&b, "<key>APFSVolumeUUID</key><string>%s</string><key>DeviceIdentifier</key><string>%s</string>", v.APFSVolumeUUID, v.DeviceIdentifier)
		b.WriteString("<key>Roles</key><array>")
		for _, r := range v.Roles {
			fmt.Fprintf(&b, "<string>%s</string>", r)
		}
		b.WriteString("</array></dict>\n")
	}
	b.WriteString("\t\t\t</array>\n\t\t</dict>\n\t</array>\n</dict>\n</plist>")
	return []byte(b.String())
}

func TestGenerateCryptoUsers(t *testing.T) {
	localUser := cryptoUser{APFSCryptoUserType: "LocalOpenDirectory", APFSCryptoUserUUID: "3C7DB8EE-5EA4-4C18-A7E4-25C6C0F94667", VolumeOwner: true}
	recovery := cryptoUser{APFSCryptoUserType: "PersonalRecovery", APFSCryptoUserUUID: "EBC6C064-0000-11AA-AA11-00306543ECAC", VolumeOwner: true}
	mdm := cryptoUser{APFSCryptoUserType: "MDMRecovery", APFSCryptoUserUUID: "2457711A-523C-4604-B75A-F48A571D5036", VolumeOwner: false}

	system := Volume{DeviceIdentifier: "disk3s1", APFSVolumeUUID: "sys-uuid", Roles: []string{"System"}}
	preboot := Volume{DeviceIdentifier: "disk3s2", APFSVolumeUUID: "pre-uuid", Roles: []string{"Preboot"}}
	data := Volume{DeviceIdentifier: "disk3s5", APFSVolumeUUID: "data-uuid", Roles: []string{"Data"}}

	t.Run("queries only the data volume when present", func(t *testing.T) {
		queried := make([]string, 0)
		rows, err := generateCryptoUsers(listPlist(t, system, preboot, data), func(device string) ([]byte, error) {
			queried = append(queried, device)
			return cryptoUsersPlist(t, localUser, recovery, mdm), nil
		})
		require.NoError(t, err)
		require.Equal(t, []string{"disk3s5"}, queried) // System/Preboot are not queried
		require.Equal(t, []map[string]string{
			{"device_identifier": "disk3s5", "volume_uuid": "data-uuid", "crypto_user_uuid": localUser.APFSCryptoUserUUID, "type": "LocalOpenDirectory", "volume_owner": "1"},
			{"device_identifier": "disk3s5", "volume_uuid": "data-uuid", "crypto_user_uuid": recovery.APFSCryptoUserUUID, "type": "PersonalRecovery", "volume_owner": "1"},
			{"device_identifier": "disk3s5", "volume_uuid": "data-uuid", "crypto_user_uuid": mdm.APFSCryptoUserUUID, "type": "MDMRecovery", "volume_owner": "0"},
		}, rows)
	})

	t.Run("falls back to all volumes and dedups by crypto user", func(t *testing.T) {
		rows, err := generateCryptoUsers(listPlist(t, system, preboot), func(device string) ([]byte, error) {
			switch device {
			case "disk3s1":
				return cryptoUsersPlist(t, localUser, recovery), nil
			case "disk3s2":
				return cryptoUsersPlist(t, localUser, mdm), nil // localUser overlaps disk3s1
			default:
				return cryptoUsersPlist(t), nil
			}
		})
		require.NoError(t, err)
		require.Equal(t, []map[string]string{
			{"device_identifier": "disk3s1", "volume_uuid": "sys-uuid", "crypto_user_uuid": localUser.APFSCryptoUserUUID, "type": "LocalOpenDirectory", "volume_owner": "1"},
			{"device_identifier": "disk3s1", "volume_uuid": "sys-uuid", "crypto_user_uuid": recovery.APFSCryptoUserUUID, "type": "PersonalRecovery", "volume_owner": "1"},
			{"device_identifier": "disk3s2", "volume_uuid": "pre-uuid", "crypto_user_uuid": mdm.APFSCryptoUserUUID, "type": "MDMRecovery", "volume_owner": "0"},
		}, rows)
	})

	t.Run("skips a volume whose listCryptoUsers call fails", func(t *testing.T) {
		rows, err := generateCryptoUsers(listPlist(t, data), func(device string) ([]byte, error) {
			return nil, fmt.Errorf("diskutil: %s could not be found", device)
		})
		require.NoError(t, err)
		require.Empty(t, rows)
	})

	t.Run("skips a volume with malformed listCryptoUsers output", func(t *testing.T) {
		rows, err := generateCryptoUsers(listPlist(t, data), func(device string) ([]byte, error) {
			return []byte("not a plist"), nil
		})
		require.NoError(t, err)
		require.Empty(t, rows)
	})

	t.Run("returns error on malformed list output", func(t *testing.T) {
		_, err := generateCryptoUsers([]byte("not a plist"), func(device string) ([]byte, error) {
			return nil, nil
		})
		require.Error(t, err)
	})
}
