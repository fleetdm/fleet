package service

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// plistOpt returns a pointer to queryResponses[key] cast to T, or nil if the
// key is absent or holds a value of a different type. Mirrors the existing
// "only assign when present" pattern in handleRefetchDeviceResults, extended
// to the generic case via Go generics since T varies per Apple query key
// (string, bool, float64, int64, time.Time).
func plistOpt[T any](queryResponses map[string]any, key string) *T {
	v, ok := queryResponses[key]
	if !ok {
		return nil
	}
	t, ok := v.(T)
	if !ok {
		return nil
	}
	return &t
}

// plistOptInt64 returns queryResponses[key] as an int64, or nil if absent/wrong
// type. The plist library (github.com/micromdm/plist) decodes a plist
// <integer> into either int64 or uint64 depending on whether the value is
// negative (see its signedInt type) — non-negative values, the common case
// for Apple's DeviceInformation integer fields (CellularTechnology, the
// AccessibilitySettings TextSize), decode as uint64 even in XML plists, and
// binary plists decode ALL integers as uint64 regardless of sign. Asserting
// only int64 would silently drop every such value.
func plistOptInt64(queryResponses map[string]any, key string) *int64 {
	v, ok := queryResponses[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case int64:
		return &n
	case uint64:
		i := int64(n) //nolint:gosec // Apple's documented integer fields fit comfortably in int64.
		return &i
	default:
		return nil
	}
}

// plistOptBytes returns queryResponses[key] cast to []byte (the Go type the
// plist library produces for a plist <data> element), or nil if absent/wrong
// type.
func plistOptBytes(queryResponses map[string]any, key string) []byte {
	v, ok := queryResponses[key]
	if !ok {
		return nil
	}
	b, ok := v.([]byte)
	if !ok {
		return nil
	}
	return b
}

// plistOptDataArray returns queryResponses[key] cast to [][]byte (an array of
// plist <data> elements), or nil if absent/wrong type. Used for
// DevicePropertiesAttestation, whose value is a DER certificate chain.
func plistOptDataArray(queryResponses map[string]any, key string) [][]byte {
	v, ok := queryResponses[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([][]byte, 0, len(arr))
	for _, elem := range arr {
		if b, ok := elem.([]byte); ok {
			out = append(out, b)
		}
	}
	return out
}

// plistOptDictArray returns queryResponses[key] cast to a slice of nested
// dictionaries, or nil if absent/wrong type. Used for ServiceSubscriptions.
func plistOptDictArray(queryResponses map[string]any, key string) []map[string]any {
	v, ok := queryResponses[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, elem := range arr {
		if m, ok := elem.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func parseMDMAppleAccessibilitySettings(queryResponses map[string]any) *fleet.MDMAppleAccessibilitySettings {
	m, ok := queryResponses["AccessibilitySettings"].(map[string]any)
	if !ok {
		return nil
	}
	return &fleet.MDMAppleAccessibilitySettings{
		BoldTextEnabled:            plistOpt[bool](m, "BoldTextEnabled"),
		GrayscaleEnabled:           plistOpt[bool](m, "GrayscaleEnabled"),
		IncreaseContrastEnabled:    plistOpt[bool](m, "IncreaseContrastEnabled"),
		ReduceMotionEnabled:        plistOpt[bool](m, "ReduceMotionEnabled"),
		ReduceTransparencyEnabled:  plistOpt[bool](m, "ReduceTransparencyEnabled"),
		TextSize:                   plistOptInt64(m, "TextSize"),
		TouchAccommodationsEnabled: plistOpt[bool](m, "TouchAccommodationsEnabled"),
		VoiceOverEnabled:           plistOpt[bool](m, "VoiceOverEnabled"),
		ZoomEnabled:                plistOpt[bool](m, "ZoomEnabled"),
	}
}

func parseMDMAppleOrganizationInfo(queryResponses map[string]any) *fleet.MDMAppleOrganizationInfo {
	m, ok := queryResponses["OrganizationInfo"].(map[string]any)
	if !ok {
		return nil
	}
	return &fleet.MDMAppleOrganizationInfo{
		OrganizationName:    plistOpt[string](m, "OrganizationName"),
		OrganizationAddress: plistOpt[string](m, "OrganizationAddress"),
		OrganizationPhone:   plistOpt[string](m, "OrganizationPhone"),
		OrganizationEmail:   plistOpt[string](m, "OrganizationEmail"),
		OrganizationMagic:   plistOpt[string](m, "OrganizationMagic"),
	}
}

func parseMDMAppleDeviceVitalsMDMOptions(queryResponses map[string]any) *fleet.MDMAppleDeviceVitalsMDMOptions {
	m, ok := queryResponses["MDMOptions"].(map[string]any)
	if !ok {
		return nil
	}
	return &fleet.MDMAppleDeviceVitalsMDMOptions{
		ActivationLockAllowedWhileSupervised:             plistOpt[bool](m, "ActivationLockAllowedWhileSupervised"),
		BootstrapTokenAllowed:                            plistOpt[bool](m, "BootstrapTokenAllowed"),
		PromptUserToAllowBootstrapTokenForAuthentication: plistOpt[bool](m, "PromptUserToAllowBootstrapTokenForAuthentication"),
	}
}

func parseMDMAppleServiceSubscriptions(queryResponses map[string]any) []fleet.MDMAppleServiceSubscription {
	dicts := plistOptDictArray(queryResponses, "ServiceSubscriptions")
	if dicts == nil {
		return nil
	}
	subscriptions := make([]fleet.MDMAppleServiceSubscription, 0, len(dicts))
	for _, m := range dicts {
		slot := plistOpt[string](m, "Slot")
		if slot == nil {
			// Slot is the child table's key; a subscription entry without one can't
			// be persisted.
			continue
		}
		subscriptions = append(subscriptions, fleet.MDMAppleServiceSubscription{
			Slot:                     *slot,
			CarrierSettingsVersion:   plistOpt[string](m, "CarrierSettingsVersion"),
			CurrentCarrierNetwork:    plistOpt[string](m, "CurrentCarrierNetwork"),
			CurrentMCC:               plistOpt[string](m, "CurrentMCC"),
			CurrentMNC:               plistOpt[string](m, "CurrentMNC"),
			EID:                      plistOpt[string](m, "EID"),
			ICCID:                    plistOpt[string](m, "ICCID"),
			IMEI:                     plistOpt[string](m, "IMEI"),
			IsDataPreferred:          plistOpt[bool](m, "IsDataPreferred"),
			IsRoaming:                plistOpt[bool](m, "IsRoaming"),
			IsVoicePreferred:         plistOpt[bool](m, "IsVoicePreferred"),
			Label:                    plistOpt[string](m, "Label"),
			LabelID:                  plistOpt[string](m, "LabelID"),
			MEID:                     plistOpt[string](m, "MEID"),
			PhoneNumber:              plistOpt[string](m, "PhoneNumber"),
			SubscriberCarrierNetwork: plistOpt[string](m, "SubscriberCarrierNetwork"),
		})
	}
	return subscriptions
}

// parseMDMAppleDeviceVitals builds the fields to persist to
// host_mdm_apple_device_vitals / host_mdm_apple_service_subscriptions from a
// DeviceInformation command ack's QueryResponses dictionary. Fields absent
// from queryResponses (an enrollment method that doesn't support them, or an
// older OS) are left nil, which SetOrUpdateHostMDMAppleDeviceVitals persists
// as SQL NULL.
func parseMDMAppleDeviceVitals(queryResponses map[string]any) fleet.MDMAppleDeviceVitals {
	return fleet.MDMAppleDeviceVitals{
		UDID:                       plistOpt[string](queryResponses, "UDID"),
		ModelNumber:                plistOpt[string](queryResponses, "ModelNumber"),
		ModemFirmwareVersion:       plistOpt[string](queryResponses, "ModemFirmwareVersion"),
		SupplementalBuildVersion:   plistOpt[string](queryResponses, "SupplementalBuildVersion"),
		SupplementalOSVersionExtra: plistOpt[string](queryResponses, "SupplementalOSVersionExtra"),
		BluetoothMAC:               plistOpt[string](queryResponses, "BluetoothMAC"),
		WiFiMAC:                    plistOpt[string](queryResponses, "WiFiMAC"),
		EASDeviceIdentifier:        plistOpt[string](queryResponses, "EASDeviceIdentifier"),
		ITunesStoreAccountHash:     plistOpt[string](queryResponses, "iTunesStoreAccountHash"),
		PushToken:                  plistOptBytes(queryResponses, "PushToken"),

		BatteryLevel:       plistOpt[float64](queryResponses, "BatteryLevel"),
		CellularTechnology: plistOptInt64(queryResponses, "CellularTechnology"),

		AppAnalyticsEnabled:           plistOpt[bool](queryResponses, "AppAnalyticsEnabled"),
		AwaitingConfiguration:         plistOpt[bool](queryResponses, "AwaitingConfiguration"),
		DataRoamingEnabled:            plistOpt[bool](queryResponses, "DataRoamingEnabled"),
		DiagnosticSubmissionEnabled:   plistOpt[bool](queryResponses, "DiagnosticSubmissionEnabled"),
		IsCloudBackupEnabled:          plistOpt[bool](queryResponses, "IsCloudBackupEnabled"),
		IsDeviceLocatorServiceEnabled: plistOpt[bool](queryResponses, "IsDeviceLocatorServiceEnabled"),
		IsDoNotDisturbInEffect:        plistOpt[bool](queryResponses, "IsDoNotDisturbInEffect"),
		IsMDMLostModeEnabled:          plistOpt[bool](queryResponses, "IsMDMLostModeEnabled"),
		IsNetworkTethered:             plistOpt[bool](queryResponses, "IsNetworkTethered"),
		ITunesStoreAccountIsActive:    plistOpt[bool](queryResponses, "iTunesStoreAccountIsActive"),
		PersonalHotspotEnabled:        plistOpt[bool](queryResponses, "PersonalHotspotEnabled"),

		LastCloudBackupDate: plistOpt[time.Time](queryResponses, "LastCloudBackupDate"),

		AccessibilitySettings:       parseMDMAppleAccessibilitySettings(queryResponses),
		OrganizationInfo:            parseMDMAppleOrganizationInfo(queryResponses),
		MDMOptions:                  parseMDMAppleDeviceVitalsMDMOptions(queryResponses),
		DevicePropertiesAttestation: plistOptDataArray(queryResponses, "DevicePropertiesAttestation"),

		ServiceSubscriptions: parseMDMAppleServiceSubscriptions(queryResponses),
	}
}
