package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// androidDeviceVitalsRow mirrors host_mdm_android_device_vitals' columns for
// named-parameter binding. The JSON columns are pre-marshaled ([]byte) since
// sqlx does not marshal nested slices on its own; a nil []byte binds as SQL
// NULL.
type androidDeviceVitalsRow struct {
	HostUUID string `db:"host_uuid"`

	AdbEnabled         *bool `db:"adb_enabled"`
	PasscodeProtected  *bool `db:"passcode_protected"`
	PlayProtectEnabled *bool `db:"play_protect_enabled"`

	EncryptionType        *string `db:"encryption_type"`
	Manufacturer          *string `db:"manufacturer"`
	SecurityUpdateVersion *string `db:"security_update_version"`
	DeviceKernelVersion   *string `db:"device_kernel_version"`
	BootloaderVersion     *string `db:"bootloader_version"`
	SystemUpdateStatus    *string `db:"system_update_status"`
	SecurityPosture       *string `db:"security_posture"`
	IMEI                  *string `db:"imei"`
	MEID                  *string `db:"meid"`

	APILevel *int64 `db:"api_level"`

	SecurityPostureDetails []byte `db:"security_posture_details"`
	TelephonyInfos         []byte `db:"telephony_infos"`
}

func newAndroidDeviceVitalsRow(ctx context.Context, hostUUID string, vitals fleet.MDMAndroidDeviceVitals) (*androidDeviceVitalsRow, error) {
	row := &androidDeviceVitalsRow{
		HostUUID: hostUUID,

		AdbEnabled:         vitals.AdbEnabled,
		PasscodeProtected:  vitals.PasscodeProtected,
		PlayProtectEnabled: vitals.PlayProtectEnabled,

		EncryptionType:        vitals.EncryptionType,
		Manufacturer:          vitals.Manufacturer,
		SecurityUpdateVersion: vitals.SecurityUpdateVersion,
		DeviceKernelVersion:   vitals.DeviceKernelVersion,
		BootloaderVersion:     vitals.BootloaderVersion,
		SystemUpdateStatus:    vitals.SystemUpdateStatus,
		SecurityPosture:       vitals.SecurityPosture,
		IMEI:                  vitals.IMEI,
		MEID:                  vitals.MEID,

		APILevel: vitals.APILevel,
	}

	var err error
	if row.SecurityPostureDetails, err = jsonColumn(vitals.SecurityPostureDetails); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal security posture details")
	}
	if row.TelephonyInfos, err = jsonColumn(vitals.TelephonyInfos); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal telephony infos")
	}
	return row, nil
}

// SetOrUpdateHostMDMAndroidDeviceVitals persists the Android vitals extracted
// from an AMAPI status report: an update-then-insert-on-no-match of
// host_mdm_android_device_vitals (most status reports are updates after the
// first).
//
// A status report is a full snapshot of the device, so this is a full
// overwrite: a section the device stops reporting (deviceSettings once the
// applied policy turns deviceSettingsEnabled off, say) nulls its columns
// rather than leaving a stale value on the host.
//
// The update-then-insert relies on clientFoundRows being set on the DSN (see
// the mysql config in server/platform/mysql): without it, re-reporting
// identical vitals would report 0 rows affected and fall through to an insert
// that violates the primary key. Two concurrent first reports for the same
// host can still race into a duplicate-key error on the insert; Pub/Sub
// redelivery makes that self-healing.
func (ds *Datastore) SetOrUpdateHostMDMAndroidDeviceVitals(ctx context.Context, hostUUID string, vitals fleet.MDMAndroidDeviceVitals) error {
	const updateStmt = `
		UPDATE host_mdm_android_device_vitals SET
			adb_enabled = :adb_enabled,
			passcode_protected = :passcode_protected,
			play_protect_enabled = :play_protect_enabled,
			encryption_type = :encryption_type,
			manufacturer = :manufacturer,
			security_update_version = :security_update_version,
			device_kernel_version = :device_kernel_version,
			bootloader_version = :bootloader_version,
			system_update_status = :system_update_status,
			security_posture = :security_posture,
			imei = :imei,
			meid = :meid,
			api_level = :api_level,
			security_posture_details = :security_posture_details,
			telephony_infos = :telephony_infos
		WHERE host_uuid = :host_uuid`

	const insertStmt = `
		INSERT INTO host_mdm_android_device_vitals (
			host_uuid, adb_enabled, passcode_protected, play_protect_enabled, encryption_type,
			manufacturer, security_update_version, device_kernel_version, bootloader_version,
			system_update_status, security_posture, imei, meid, api_level,
			security_posture_details, telephony_infos
		) VALUES (
			:host_uuid, :adb_enabled, :passcode_protected, :play_protect_enabled, :encryption_type,
			:manufacturer, :security_update_version, :device_kernel_version, :bootloader_version,
			:system_update_status, :security_posture, :imei, :meid, :api_level,
			:security_posture_details, :telephony_infos
		)`

	row, err := newAndroidDeviceVitalsRow(ctx, hostUUID, vitals)
	if err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		result, err := sqlx.NamedExecContext(ctx, tx, updateStmt, row)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update host mdm android device vitals")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			if _, err := sqlx.NamedExecContext(ctx, tx, insertStmt, row); err != nil {
				return ctxerr.Wrap(ctx, err, "insert host mdm android device vitals")
			}
		}
		return nil
	})
}

const androidDeviceVitalsSelectStmt = `
	SELECT
		host_uuid, adb_enabled, passcode_protected, play_protect_enabled, encryption_type,
		manufacturer, security_update_version, device_kernel_version, bootloader_version,
		system_update_status, security_posture, imei, meid, api_level,
		security_posture_details, telephony_infos
	FROM host_mdm_android_device_vitals
	WHERE host_uuid = ?`

func (ds *Datastore) LoadHostMDMAndroidDeviceVitals(ctx context.Context, host *fleet.Host) error {
	var row androidDeviceVitalsRow
	err := sqlx.GetContext(ctx, ds.reader(ctx), &row, androidDeviceVitalsSelectStmt, host.UUID)
	switch {
	case err == nil:
		host.AdbEnabled = row.AdbEnabled
		host.PasscodeProtected = row.PasscodeProtected
		host.PlayProtectEnabled = row.PlayProtectEnabled
		host.EncryptionType = row.EncryptionType
		host.Manufacturer = row.Manufacturer
		host.SecurityUpdateVersion = row.SecurityUpdateVersion
		host.DeviceKernelVersion = row.DeviceKernelVersion
		host.BootloaderVersion = row.BootloaderVersion
		host.SystemUpdateStatus = row.SystemUpdateStatus
		host.SecurityPosture = row.SecurityPosture
		host.IMEI = row.IMEI
		host.MEID = row.MEID
		host.APILevel = row.APILevel

		if row.SecurityPostureDetails != nil {
			if err := json.Unmarshal(row.SecurityPostureDetails, &host.SecurityPostureDetails); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host security posture details")
			}
		}
		if row.TelephonyInfos != nil {
			if err := json.Unmarshal(row.TelephonyInfos, &host.TelephonyInfos); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host telephony infos")
			}
		}
	case errors.Is(err, sql.ErrNoRows):
		// no vitals collected yet for this host; leave fields nil.
	default:
		return ctxerr.Wrap(ctx, err, "get host mdm android device vitals")
	}

	return nil
}
