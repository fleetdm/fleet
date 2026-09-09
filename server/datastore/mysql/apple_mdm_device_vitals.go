package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// deviceVitalsRow mirrors host_mdm_apple_device_vitals' columns for named-
// parameter binding. The JSON columns are pre-marshaled ([]byte) since sqlx
// does not marshal nested structs on its own; a nil []byte binds as SQL NULL.
type deviceVitalsRow struct {
	HostUUID string `db:"host_uuid"`

	UDID                       *string `db:"udid"`
	ModelNumber                *string `db:"model_number"`
	ModemFirmwareVersion       *string `db:"modem_firmware_version"`
	SupplementalBuildVersion   *string `db:"supplemental_build_version"`
	SupplementalOSVersionExtra *string `db:"supplemental_os_version_extra"`
	BluetoothMAC               *string `db:"bluetooth_mac"`
	WiFiMAC                    *string `db:"wifi_mac"`
	EASDeviceIdentifier        *string `db:"eas_device_identifier"`
	ITunesStoreAccountHash     *string `db:"itunes_store_account_hash"`
	PushToken                  []byte  `db:"push_token"`

	BatteryLevel       *float64 `db:"battery_level"`
	CellularTechnology *int64   `db:"cellular_technology"`

	AppAnalyticsEnabled           *bool `db:"app_analytics_enabled"`
	AwaitingConfiguration         *bool `db:"awaiting_configuration"`
	DataRoamingEnabled            *bool `db:"data_roaming_enabled"`
	DiagnosticSubmissionEnabled   *bool `db:"diagnostic_submission_enabled"`
	IsCloudBackupEnabled          *bool `db:"is_cloud_backup_enabled"`
	IsDeviceLocatorServiceEnabled *bool `db:"is_device_locator_service_enabled"`
	IsDoNotDisturbInEffect        *bool `db:"is_do_not_disturb_in_effect"`
	IsMDMLostModeEnabled          *bool `db:"is_mdm_lost_mode_enabled"`
	IsNetworkTethered             *bool `db:"is_network_tethered"`
	ITunesStoreAccountIsActive    *bool `db:"itunes_store_account_is_active"`
	PersonalHotspotEnabled        *bool `db:"personal_hotspot_enabled"`

	LastCloudBackupDate *time.Time `db:"last_cloud_backup_date"`

	AccessibilitySettings       []byte `db:"accessibility_settings"`
	OrganizationInfo            []byte `db:"organization_info"`
	MDMOptions                  []byte `db:"mdm_options"`
	DevicePropertiesAttestation []byte `db:"device_properties_attestation"`
}

// jsonColumn marshals v to JSON, except when v is a nil pointer or nil slice,
// in which case it returns a nil []byte (bound as SQL NULL) rather than the
// JSON literal "null".
func jsonColumn(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || ((rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Slice) && rv.IsNil()) {
		return nil, nil
	}
	return json.Marshal(v)
}

func newDeviceVitalsRow(ctx context.Context, hostUUID string, vitals fleet.MDMAppleDeviceVitals) (*deviceVitalsRow, error) {
	row := &deviceVitalsRow{
		HostUUID: hostUUID,

		UDID:                       vitals.UDID,
		ModelNumber:                vitals.ModelNumber,
		ModemFirmwareVersion:       vitals.ModemFirmwareVersion,
		SupplementalBuildVersion:   vitals.SupplementalBuildVersion,
		SupplementalOSVersionExtra: vitals.SupplementalOSVersionExtra,
		BluetoothMAC:               vitals.BluetoothMAC,
		WiFiMAC:                    vitals.WiFiMAC,
		EASDeviceIdentifier:        vitals.EASDeviceIdentifier,
		ITunesStoreAccountHash:     vitals.ITunesStoreAccountHash,
		PushToken:                  vitals.PushToken,

		BatteryLevel:       vitals.BatteryLevel,
		CellularTechnology: vitals.CellularTechnology,

		AppAnalyticsEnabled:           vitals.AppAnalyticsEnabled,
		AwaitingConfiguration:         vitals.AwaitingConfiguration,
		DataRoamingEnabled:            vitals.DataRoamingEnabled,
		DiagnosticSubmissionEnabled:   vitals.DiagnosticSubmissionEnabled,
		IsCloudBackupEnabled:          vitals.IsCloudBackupEnabled,
		IsDeviceLocatorServiceEnabled: vitals.IsDeviceLocatorServiceEnabled,
		IsDoNotDisturbInEffect:        vitals.IsDoNotDisturbInEffect,
		IsMDMLostModeEnabled:          vitals.IsMDMLostModeEnabled,
		IsNetworkTethered:             vitals.IsNetworkTethered,
		ITunesStoreAccountIsActive:    vitals.ITunesStoreAccountIsActive,
		PersonalHotspotEnabled:        vitals.PersonalHotspotEnabled,

		LastCloudBackupDate: vitals.LastCloudBackupDate,
	}

	var err error
	if row.AccessibilitySettings, err = jsonColumn(vitals.AccessibilitySettings); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal accessibility settings")
	}
	if row.OrganizationInfo, err = jsonColumn(vitals.OrganizationInfo); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal organization info")
	}
	if row.MDMOptions, err = jsonColumn(vitals.MDMOptions); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal mdm options")
	}
	if row.DevicePropertiesAttestation, err = jsonColumn(vitals.DevicePropertiesAttestation); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal device properties attestation")
	}
	return row, nil
}

// SetOrUpdateHostMDMAppleDeviceVitals persists the iOS/iPadOS vitals parsed
// from a DeviceInformation command ack: an update-then-insert-on-no-match of
// host_mdm_apple_device_vitals (most refetches are updates after the first),
// plus a replace of the host's host_mdm_apple_service_subscriptions rows, in
// a single transaction.
func (ds *Datastore) SetOrUpdateHostMDMAppleDeviceVitals(ctx context.Context, hostUUID string, vitals fleet.MDMAppleDeviceVitals) error {
	const updateStmt = `
		UPDATE host_mdm_apple_device_vitals SET
			udid = :udid,
			model_number = :model_number,
			modem_firmware_version = :modem_firmware_version,
			supplemental_build_version = :supplemental_build_version,
			supplemental_os_version_extra = :supplemental_os_version_extra,
			bluetooth_mac = :bluetooth_mac,
			wifi_mac = :wifi_mac,
			eas_device_identifier = :eas_device_identifier,
			itunes_store_account_hash = :itunes_store_account_hash,
			push_token = :push_token,
			battery_level = :battery_level,
			cellular_technology = :cellular_technology,
			app_analytics_enabled = :app_analytics_enabled,
			awaiting_configuration = :awaiting_configuration,
			data_roaming_enabled = :data_roaming_enabled,
			diagnostic_submission_enabled = :diagnostic_submission_enabled,
			is_cloud_backup_enabled = :is_cloud_backup_enabled,
			is_device_locator_service_enabled = :is_device_locator_service_enabled,
			is_do_not_disturb_in_effect = :is_do_not_disturb_in_effect,
			is_mdm_lost_mode_enabled = :is_mdm_lost_mode_enabled,
			is_network_tethered = :is_network_tethered,
			itunes_store_account_is_active = :itunes_store_account_is_active,
			personal_hotspot_enabled = :personal_hotspot_enabled,
			last_cloud_backup_date = :last_cloud_backup_date,
			accessibility_settings = :accessibility_settings,
			organization_info = :organization_info,
			mdm_options = :mdm_options,
			device_properties_attestation = :device_properties_attestation
		WHERE host_uuid = :host_uuid`

	const insertStmt = `
		INSERT INTO host_mdm_apple_device_vitals (
			host_uuid, udid, model_number, modem_firmware_version, supplemental_build_version,
			supplemental_os_version_extra, bluetooth_mac, wifi_mac, eas_device_identifier,
			itunes_store_account_hash, push_token, battery_level, cellular_technology,
			app_analytics_enabled, awaiting_configuration, data_roaming_enabled,
			diagnostic_submission_enabled, is_cloud_backup_enabled, is_device_locator_service_enabled,
			is_do_not_disturb_in_effect, is_mdm_lost_mode_enabled, is_network_tethered,
			itunes_store_account_is_active, personal_hotspot_enabled, last_cloud_backup_date,
			accessibility_settings, organization_info, mdm_options, device_properties_attestation
		) VALUES (
			:host_uuid, :udid, :model_number, :modem_firmware_version, :supplemental_build_version,
			:supplemental_os_version_extra, :bluetooth_mac, :wifi_mac, :eas_device_identifier,
			:itunes_store_account_hash, :push_token, :battery_level, :cellular_technology,
			:app_analytics_enabled, :awaiting_configuration, :data_roaming_enabled,
			:diagnostic_submission_enabled, :is_cloud_backup_enabled, :is_device_locator_service_enabled,
			:is_do_not_disturb_in_effect, :is_mdm_lost_mode_enabled, :is_network_tethered,
			:itunes_store_account_is_active, :personal_hotspot_enabled, :last_cloud_backup_date,
			:accessibility_settings, :organization_info, :mdm_options, :device_properties_attestation
		)`

	row, err := newDeviceVitalsRow(ctx, hostUUID, vitals)
	if err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		result, err := sqlx.NamedExecContext(ctx, tx, updateStmt, row)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update host mdm apple device vitals")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			if _, err := sqlx.NamedExecContext(ctx, tx, insertStmt, row); err != nil {
				return ctxerr.Wrap(ctx, err, "insert host mdm apple device vitals")
			}
		}

		return replaceHostMDMAppleServiceSubscriptions(ctx, tx, hostUUID, vitals.ServiceSubscriptions)
	})
}

func deleteHostMDMAppleDeviceVitalsDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM host_mdm_apple_device_vitals WHERE host_uuid = ?`, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host mdm apple device vitals")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ?`, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host mdm apple service subscriptions")
	}
	return nil
}

// replaceHostMDMAppleServiceSubscriptions replaces the host's service
// subscription rows to match subscriptions: rows for slots no longer
// present are deleted, current slots are upserted. Modeled on
// ReplaceHostBatteries — the number of subscriptions per host is small
// (dual-SIM at most), so a full diff-and-replace per call is cheap.
func replaceHostMDMAppleServiceSubscriptions(ctx context.Context, tx sqlx.ExtContext, hostUUID string, subscriptions []fleet.MDMAppleServiceSubscription) error {
	var existingSlots []string
	if err := sqlx.SelectContext(ctx, tx, &existingSlots,
		`SELECT slot FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ?`, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "select existing host service subscription slots")
	}

	currentSlots := make(map[string]struct{}, len(subscriptions))
	for _, s := range subscriptions {
		currentSlots[s.Slot] = struct{}{}
	}

	var staleSlots []string
	for _, slot := range existingSlots {
		if _, ok := currentSlots[slot]; !ok {
			staleSlots = append(staleSlots, slot)
		}
	}
	if len(staleSlots) > 0 {
		stmt, args, err := sqlx.In(
			`DELETE FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ? AND slot IN (?)`,
			hostUUID, staleSlots)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete stale host service subscriptions")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale host service subscriptions")
		}
	}

	const updateStmt = `
		UPDATE host_mdm_apple_service_subscriptions SET
			carrier_settings_version = :carrier_settings_version,
			current_carrier_network = :current_carrier_network,
			current_mcc = :current_mcc,
			current_mnc = :current_mnc,
			eid = :eid,
			iccid = :iccid,
			imei = :imei,
			is_data_preferred = :is_data_preferred,
			is_roaming = :is_roaming,
			is_voice_preferred = :is_voice_preferred,
			label = :label,
			label_id = :label_id,
			meid = :meid,
			phone_number = :phone_number,
			subscriber_carrier_network = :subscriber_carrier_network
		WHERE host_uuid = :host_uuid AND slot = :slot`

	const insertStmt = `
		INSERT INTO host_mdm_apple_service_subscriptions (
			host_uuid, slot, carrier_settings_version, current_carrier_network, current_mcc, current_mnc,
			eid, iccid, imei, is_data_preferred, is_roaming, is_voice_preferred, label, label_id, meid,
			phone_number, subscriber_carrier_network
		) VALUES (
			:host_uuid, :slot, :carrier_settings_version, :current_carrier_network, :current_mcc, :current_mnc,
			:eid, :iccid, :imei, :is_data_preferred, :is_roaming, :is_voice_preferred, :label, :label_id, :meid,
			:phone_number, :subscriber_carrier_network
		)`

	// Update-then-insert-on-no-match per slot: the row count per host is tiny
	// (dual-SIM at most), and most refetches update an existing slot.
	for _, s := range subscriptions {
		s.HostUUID = hostUUID
		result, err := sqlx.NamedExecContext(ctx, tx, updateStmt, s)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update host service subscription")
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			if _, err := sqlx.NamedExecContext(ctx, tx, insertStmt, s); err != nil {
				return ctxerr.Wrap(ctx, err, "insert host service subscription")
			}
		}
	}
	return nil
}

const deviceVitalsSelectStmt = `
	SELECT
		host_uuid, udid, model_number, modem_firmware_version, supplemental_build_version,
		supplemental_os_version_extra, bluetooth_mac, wifi_mac, eas_device_identifier,
		itunes_store_account_hash, push_token, battery_level, cellular_technology,
		app_analytics_enabled, awaiting_configuration, data_roaming_enabled,
		diagnostic_submission_enabled, is_cloud_backup_enabled, is_device_locator_service_enabled,
		is_do_not_disturb_in_effect, is_mdm_lost_mode_enabled, is_network_tethered,
		itunes_store_account_is_active, personal_hotspot_enabled, last_cloud_backup_date,
		accessibility_settings, organization_info, mdm_options, device_properties_attestation
	FROM host_mdm_apple_device_vitals
	WHERE host_uuid = ?`

const serviceSubscriptionsSelectStmt = `
	SELECT
		host_uuid, slot, carrier_settings_version, current_carrier_network, current_mcc, current_mnc,
		eid, iccid, imei, is_data_preferred, is_roaming, is_voice_preferred, label, label_id, meid,
		phone_number, subscriber_carrier_network
	FROM host_mdm_apple_service_subscriptions
	WHERE host_uuid = ?
	ORDER BY slot`

func (ds *Datastore) LoadHostMDMAppleDeviceVitals(ctx context.Context, host *fleet.Host) error {
	var row deviceVitalsRow
	err := sqlx.GetContext(ctx, ds.reader(ctx), &row, deviceVitalsSelectStmt, host.UUID)
	switch err {
	case nil:
		host.UDID = row.UDID
		host.ModelNumber = row.ModelNumber
		host.ModemFirmwareVersion = row.ModemFirmwareVersion
		host.SupplementalBuildVersion = row.SupplementalBuildVersion
		host.SupplementalOSVersionExtra = row.SupplementalOSVersionExtra
		host.BluetoothMAC = row.BluetoothMAC
		host.WiFiMAC = row.WiFiMAC
		host.EASDeviceIdentifier = row.EASDeviceIdentifier
		host.ITunesStoreAccountHash = row.ITunesStoreAccountHash
		host.PushToken = row.PushToken
		host.BatteryLevel = row.BatteryLevel
		if row.CellularTechnology != nil {
			host.CellularTechnology = new(fleet.MDMAppleCellularTechnology(*row.CellularTechnology))
		}
		host.AppAnalyticsEnabled = row.AppAnalyticsEnabled
		host.AwaitingConfiguration = row.AwaitingConfiguration
		host.DataRoamingEnabled = row.DataRoamingEnabled
		host.DiagnosticSubmissionEnabled = row.DiagnosticSubmissionEnabled
		host.IsCloudBackupEnabled = row.IsCloudBackupEnabled
		host.IsDeviceLocatorServiceEnabled = row.IsDeviceLocatorServiceEnabled
		host.IsDoNotDisturbInEffect = row.IsDoNotDisturbInEffect
		host.IsMDMLostModeEnabled = row.IsMDMLostModeEnabled
		host.IsNetworkTethered = row.IsNetworkTethered
		host.ITunesStoreAccountIsActive = row.ITunesStoreAccountIsActive
		host.PersonalHotspotEnabled = row.PersonalHotspotEnabled
		host.LastCloudBackupDate = row.LastCloudBackupDate

		if row.AccessibilitySettings != nil {
			var v fleet.MDMAppleAccessibilitySettings
			if err := json.Unmarshal(row.AccessibilitySettings, &v); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host accessibility settings")
			}
			host.AccessibilitySettings = &v
		}
		if row.OrganizationInfo != nil {
			var v fleet.MDMAppleOrganizationInfo
			if err := json.Unmarshal(row.OrganizationInfo, &v); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host organization info")
			}
			host.OrganizationInfo = &v
		}
		if row.MDMOptions != nil {
			var v fleet.MDMAppleDeviceVitalsMDMOptions
			if err := json.Unmarshal(row.MDMOptions, &v); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host mdm options")
			}
			host.MDMOptions = &v
		}
		if row.DevicePropertiesAttestation != nil {
			if err := json.Unmarshal(row.DevicePropertiesAttestation, &host.DevicePropertiesAttestation); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshal host device properties attestation")
			}
		}
	case sql.ErrNoRows:
		// no vitals collected yet for this host; leave fields nil.
	default:
		return ctxerr.Wrap(ctx, err, "get host mdm apple device vitals")
	}

	var subs []fleet.MDMAppleServiceSubscription
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &subs, serviceSubscriptionsSelectStmt, host.UUID); err != nil {
		return ctxerr.Wrap(ctx, err, "get host mdm apple service subscriptions")
	}
	host.ServiceSubscriptions = subs

	return nil
}
