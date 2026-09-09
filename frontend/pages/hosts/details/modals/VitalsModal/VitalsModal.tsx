import React from "react";

import { IHost, IHostMdmAppleServiceSubscription } from "interfaces/host";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";

import {
  buildHostVitals,
  sortHostVitals,
  IHostVitalsSources,
  VitalForSort,
} from "../../cards/Vitals/Vitals";

const baseClass = "vitals-modal";

const renderBoolean = (value?: boolean | null) => {
  if (value === undefined || value === null) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }
  return value ? "True" : "False";
};

const renderText = (value?: string | null) =>
  value ? <TooltipTruncatedText value={value} /> : DEFAULT_EMPTY_CELL_VALUE;

const renderBatteryLevel = (value?: number | null) =>
  // Apple reports -1 when the device can't determine the level.
  value === undefined || value === null || value < 0
    ? DEFAULT_EMPTY_CELL_VALUE
    : `${Math.round(value * 100)}%`;

const renderLastCloudBackupDate = (value?: string | null) =>
  value ? (
    <HumanTimeDiffWithFleetLaunchCutoff timeString={value} />
  ) : (
    DEFAULT_EMPTY_CELL_VALUE
  );

/* The nested-dict vitals (accessibility settings, organization info, MDM
 * options) all have closed schemas, so each sub-key gets a hand-written label
 * the same way the top-level vitals do — rather than exposing raw API key
 * names. These three helpers return undefined for an absent sub-key so
 * renderNestedFields drops the row entirely instead of padding the list with
 * empty values; note `false` is a real value and must survive. */
const nestedBool = (value?: boolean | null) => {
  if (value === undefined || value === null) {
    return undefined;
  }
  return value ? "True" : "False";
};

const nestedText = (value?: string | null) => value || undefined;

const nestedNumber = (value?: number | null) =>
  value === undefined || value === null ? undefined : String(value);

interface INestedField {
  label: string;
  value?: string;
}

/** Renders a nested-dict vital as aligned label/value lines. Returns the
 * empty-value text when every sub-key is absent (e.g. Apple reports
 * MDMOptions as an empty dict when MDM has never set any). */
const renderNestedFields = (fields: INestedField[]) => {
  const present = fields.filter((field) => field.value !== undefined);

  if (!present.length) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  return (
    <div className={`${baseClass}__nested`}>
      {present.map(({ label, value }) => (
        <React.Fragment key={label}>
          {/* Colon appended here rather than baked into the label strings so
              they stay reusable; matches how DataSet renders its own titles. */}
          <span className={`${baseClass}__nested-label`}>{label}:</span>
          <span className={`${baseClass}__nested-value`}>{value}</span>
        </React.Fragment>
      ))}
    </div>
  );
};

/** Renders a list-valued vital (the attestation certificate chain) as one item
 * per line. Empty entries are dropped. */
const renderLines = (values?: Array<string | null | undefined>) => {
  const lines = (values ?? []).filter((value): value is string =>
    Boolean(value)
  );

  if (!lines.length) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  return (
    <div className={`${baseClass}__lines`}>
      {lines.map((line) => (
        <div key={line}>{line}</div>
      ))}
    </div>
  );
};

/** Formats label/value pairs for a tooltip body, matching the Hardware model
 * tooltip treatment (see getHardwareModelDisplay): bold "Label:" followed by
 * the plain value, one pair per line. */
const renderTooltipFields = (fields: INestedField[]) => (
  // Left-align to override the tooltip's default centered text.
  <div style={{ textAlign: "left" }}>
    {fields
      .filter((field) => field.value !== undefined)
      .map(({ label, value }, i) => (
        <React.Fragment key={label}>
          {i > 0 && <br />}
          <b>{label}:</b> {value}
        </React.Fragment>
      ))}
  </div>
);

const subscriptionFields = (
  sub: IHostMdmAppleServiceSubscription
): INestedField[] => [
  {
    label: "Carrier settings version",
    value: nestedText(sub.carrier_settings_version),
  },
  {
    label: "Current carrier network",
    value: nestedText(sub.current_carrier_network),
  },
  { label: "Current MCC", value: nestedText(sub.current_mcc) },
  { label: "Current MNC", value: nestedText(sub.current_mnc) },
  { label: "Data preferred", value: nestedBool(sub.is_data_preferred) },
  { label: "EID", value: nestedText(sub.eid) },
  { label: "ICCID", value: nestedText(sub.iccid) },
  { label: "IMEI", value: nestedText(sub.imei) },
  { label: "Label", value: nestedText(sub.label) },
  { label: "Label ID", value: nestedText(sub.label_id) },
  { label: "MEID", value: nestedText(sub.meid) },
  { label: "Phone number", value: nestedText(sub.phone_number) },
  { label: "Roaming", value: nestedBool(sub.is_roaming) },
  { label: "Slot", value: nestedText(sub.slot) },
  {
    label: "Subscriber carrier network",
    value: nestedText(sub.subscriber_carrier_network),
  },
  { label: "Voice preferred", value: nestedBool(sub.is_voice_preferred) },
];

/** Renders one underlined "Subscription N" per cellular subscription, revealing
 * that subscription's values on hover. Dual-SIM devices report more than one,
 * and an unprovisioned eSIM slot reports only a couple of fields. */
const renderServiceSubscriptions = (
  subscriptions?: IHostMdmAppleServiceSubscription[]
) => {
  if (!subscriptions?.length) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  return (
    <div className={`${baseClass}__subscriptions`}>
      {subscriptions.map((sub, i) => (
        <TooltipWrapper
          key={sub.slot}
          tipContent={renderTooltipFields(subscriptionFields(sub))}
        >
          {/* Template literal keeps this one text node, so the label stays
              queryable as a whole string. */}
          {`Subscription ${i + 1}`}
        </TooltipWrapper>
      ))}
    </div>
  );
};

/** iOS/iPadOS vitals fields (see HostMDMAppleDeviceVitals, server/fleet/mdm_apple_device_vitals.go) */
type VitalKey =
  | "udid"
  | "model_number"
  | "modem_firmware_version"
  | "supplemental_build_version"
  | "supplemental_os_version_extra"
  | "bluetooth_mac"
  | "wifi_mac"
  | "eas_device_identifier"
  | "itunes_store_account_hash"
  | "push_token"
  | "battery_level"
  | "cellular_technology"
  | "app_analytics_enabled"
  | "awaiting_configuration"
  | "data_roaming_enabled"
  | "diagnostic_submission_enabled"
  | "is_cloud_backup_enabled"
  | "is_device_locator_service_enabled"
  | "is_do_not_disturb_in_effect"
  | "is_mdm_lost_mode_enabled"
  | "is_network_tethered"
  | "itunes_store_account_is_active"
  | "personal_hotspot_enabled"
  | "last_cloud_backup_date"
  | "accessibility_settings"
  | "organization_info"
  | "mdm_options"
  | "device_properties_attestation"
  | "service_subscriptions";

interface IVital {
  key: VitalKey;
  label: string;
  render: (host: IHost) => React.ReactNode;
  /** Multi-line values (nested field lists, line-per-entry lists) need to wrap
   * instead of truncating with an ellipsis (DataSet's default). */
  multiline?: boolean;
  tooltip?: string;
}

const VITALS: IVital[] = [
  {
    key: "accessibility_settings",
    label: "Accessibility settings",
    render: (host) => {
      const a = host.accessibility_settings;
      return renderNestedFields([
        { label: "Bold text", value: nestedBool(a?.bold_text_enabled) },
        { label: "Grayscale", value: nestedBool(a?.grayscale_enabled) },
        {
          label: "Increase contrast",
          value: nestedBool(a?.increase_contrast_enabled),
        },
        { label: "Reduce motion", value: nestedBool(a?.reduce_motion_enabled) },
        {
          label: "Reduce transparency",
          value: nestedBool(a?.reduce_transparency_enabled),
        },
        { label: "Text size", value: nestedNumber(a?.text_size) },
        {
          label: "Touch accommodations",
          value: nestedBool(a?.touch_accommodations_enabled),
        },
        { label: "VoiceOver", value: nestedBool(a?.voice_over_enabled) },
        { label: "Zoom", value: nestedBool(a?.zoom_enabled) },
      ]);
    },
    multiline: true,
  },
  {
    key: "app_analytics_enabled",
    label: "App analytics",
    render: (host) => renderBoolean(host.app_analytics_enabled),
  },
  {
    key: "awaiting_configuration",
    label: "Awaiting configuration",
    render: (host) => renderBoolean(host.awaiting_configuration),
    tooltip:
      "Determines whether the device is waiting for a Device Configured command or User Configured command to continue through Setup Assistant on the device channel or user channel, respectively.",
  },
  {
    key: "battery_level",
    label: "Battery level",
    render: (host) => renderBatteryLevel(host.battery_level),
  },
  {
    key: "bluetooth_mac",
    label: "Bluetooth MAC address",
    render: (host) => renderText(host.bluetooth_mac),
  },
  {
    key: "cellular_technology",
    label: "Cellular technology",
    render: (host) => renderText(host.cellular_technology),
  },
  {
    key: "data_roaming_enabled",
    label: "Data roaming",
    render: (host) => renderBoolean(host.data_roaming_enabled),
  },
  {
    key: "device_properties_attestation",
    label: "Device properties attestation",
    render: (host) => renderLines(host.device_properties_attestation),
    multiline: true,
  },
  {
    key: "diagnostic_submission_enabled",
    label: "Diagnostic submission",
    render: (host) => renderBoolean(host.diagnostic_submission_enabled),
  },
  {
    key: "eas_device_identifier",
    label: "EAS device identifier",
    render: (host) => renderText(host.eas_device_identifier),
    tooltip: "The device identifier for Exchange ActiveSync (EAS).",
  },
  {
    key: "is_cloud_backup_enabled",
    label: "Cloud backup enabled",
    render: (host) => renderBoolean(host.is_cloud_backup_enabled),
  },
  {
    key: "is_device_locator_service_enabled",
    label: "Device locator service enabled",
    render: (host) => renderBoolean(host.is_device_locator_service_enabled),
  },
  {
    key: "is_do_not_disturb_in_effect",
    label: "Do not disturb",
    render: (host) => renderBoolean(host.is_do_not_disturb_in_effect),
  },
  {
    key: "is_mdm_lost_mode_enabled",
    label: "Lost mode",
    render: (host) => renderBoolean(host.is_mdm_lost_mode_enabled),
  },
  {
    key: "is_network_tethered",
    label: "Network tethered",
    render: (host) => renderBoolean(host.is_network_tethered),
  },
  {
    key: "itunes_store_account_hash",
    label: "iTunes Store account hash",
    render: (host) => renderText(host.itunes_store_account_hash),
  },
  {
    key: "itunes_store_account_is_active",
    label: "iTunes Store account active",
    render: (host) => renderBoolean(host.itunes_store_account_is_active),
  },
  {
    key: "last_cloud_backup_date",
    label: "Last cloud backup",
    render: (host) => renderLastCloudBackupDate(host.last_cloud_backup_date),
  },
  {
    key: "mdm_options",
    label: "MDM options",
    render: (host) => {
      const o = host.mdm_options;
      return renderNestedFields([
        {
          label: "Activation Lock allowed while supervised",
          value: nestedBool(o?.activation_lock_allowed_while_supervised),
        },
        {
          label: "Bootstrap token allowed",
          value: nestedBool(o?.bootstrap_token_allowed),
        },
        {
          label: "Prompt user to allow bootstrap token for authentication",
          value: nestedBool(
            o?.prompt_user_to_allow_bootstrap_token_for_authentication
          ),
        },
      ]);
    },
    multiline: true,
  },
  {
    key: "model_number",
    label: "Model number",
    render: (host) => renderText(host.model_number),
  },
  {
    key: "modem_firmware_version",
    label: "Modem firmware version",
    render: (host) => renderText(host.modem_firmware_version),
  },
  {
    key: "organization_info",
    label: "Organization info",
    render: (host) => {
      const o = host.organization_info;
      return renderNestedFields([
        // Apple documents OrganizationAddress as using \n for line breaks; the
        // value cell preserves them via white-space: pre-line.
        { label: "Address", value: nestedText(o?.organization_address) },
        { label: "Email", value: nestedText(o?.organization_email) },
        { label: "Magic", value: nestedText(o?.organization_magic) },
        { label: "Name", value: nestedText(o?.organization_name) },
        { label: "Phone", value: nestedText(o?.organization_phone) },
      ]);
    },
    multiline: true,
  },
  {
    key: "personal_hotspot_enabled",
    label: "Personal hotspot",
    render: (host) => renderBoolean(host.personal_hotspot_enabled),
  },
  {
    key: "push_token",
    label: "Push token",
    render: (host) => renderText(host.push_token),
    tooltip:
      "A push token the server uses to send update notifications for a registered pass to a device.",
  },
  {
    key: "service_subscriptions",
    label: "Service subscriptions",
    render: (host) => renderServiceSubscriptions(host.service_subscriptions),
    multiline: true,
  },
  {
    key: "supplemental_build_version",
    label: "Supplemental build version",
    render: (host) => renderText(host.supplemental_build_version),
    tooltip:
      "The build version for the currently installed Background Security Improvement. If there’s no installed Background Security Improvement, this value is the same as “Build”.",
  },
  {
    key: "supplemental_os_version_extra",
    label: "Supplemental OS version extra",
    render: (host) => renderText(host.supplemental_os_version_extra),
    tooltip:
      "The OS update Background Security Improvement version letter, listed after the OS (e.g. iOS 26.3.1 (a)).",
  },
  {
    key: "udid",
    label: "UDID",
    render: (host) => renderText(host.udid),
  },
  {
    key: "wifi_mac",
    label: "Wi-Fi MAC address",
    render: (host) => renderText(host.wifi_mac),
  },
];

/** Takes the same vitals sources as the Vitals card so it can rebuild the
 * pre-existing rows, plus the full host for the iOS/iPadOS-only fields (which
 * aren't part of the card's narrower vitalsData pick). */
interface IVitalsModal extends IHostVitalsSources {
  host: IHost;
  onExit: () => void;
}

const VitalsModal = ({ host, onExit, ...vitalsSources }: IVitalsModal) => {
  const iosOnlyVitals: VitalForSort[] = VITALS.map(
    ({ key, label, render, multiline, tooltip }) => {
      return {
        sortKey: label,
        element: (
          <DataSet
            key={key}
            title={
              tooltip ? (
                <TooltipWrapper tipContent={tooltip}>{label}</TooltipWrapper>
              ) : (
                label
              )
            }
            value={render(host)}
            multiline={multiline}
          />
        ),
      };
    }
  );

  // The modal is the only place the full set is visible, so it shows the
  // pre-existing vitals (which the iOS/iPadOS card now trims to 8) alongside
  // the iOS/iPadOS-only ones, in one alphabetical list.
  const allVitals = sortHostVitals([
    ...buildHostVitals(vitalsSources),
    ...iosOnlyVitals,
  ]);

  return (
    <Modal title="Vitals" className={baseClass} onExit={onExit} width="large">
      <>
        <dl className={`${baseClass}__vitals`}>
          {allVitals.map((vital) => vital.element)}
        </dl>
        <ModalFooter
          primaryButtons={
            <Button type="button" onClick={onExit}>
              Done
            </Button>
          }
        />
      </>
    </Modal>
  );
};

export default VitalsModal;
