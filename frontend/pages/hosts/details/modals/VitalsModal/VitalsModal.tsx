import React from "react";

import { IHost } from "interfaces/host";
import { syntaxHighlight } from "utilities/helpers";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";

import {
  IosOrIpadosEnrollmentStatus,
  NOT_SUPPORTED_VITAL_TOOLTIP,
  UNSUPPORTED_VITALS_BY_ENROLLMENT,
  VitalKey,
} from "./unsupportedVitalsByEnrollment";

const baseClass = "vitals-modal";
const EMPTY_VITAL_VALUE = "None";

const renderBoolean = (value?: boolean | null) => {
  if (value === undefined || value === null) {
    return EMPTY_VITAL_VALUE;
  }
  return value ? "True" : "False";
};

const renderText = (value?: string | null) => value || EMPTY_VITAL_VALUE;

const renderBatteryLevel = (value?: number | null) =>
  value === undefined || value === null
    ? EMPTY_VITAL_VALUE
    : `${Math.round(value * 100)}%`;

const renderLastCloudBackupDate = (value?: string | null) =>
  value ? (
    <HumanTimeDiffWithFleetLaunchCutoff timeString={value} />
  ) : (
    EMPTY_VITAL_VALUE
  );

/** Renders a single-object vital (e.g. accessibility_settings) as a
 * read-only JSON preview, same treatment as PreviewPayloadModal. */
const renderJson = (value: unknown) => {
  const isEmpty =
    value === undefined ||
    value === null ||
    (typeof value === "object" && Object.keys(value).length === 0);

  if (isEmpty) {
    return EMPTY_VITAL_VALUE;
  }

  return (
    <pre
      className={`${baseClass}__json-preview`}
      // eslint-disable-next-line react/no-danger
      dangerouslySetInnerHTML={{ __html: syntaxHighlight(value) }}
    />
  );
};

/** Renders a list-valued vital as one item per line, instead of a JSON
 * preview — falsy entries (e.g. a subscription with no slot) are dropped. */
const renderLines = (values?: Array<string | null | undefined>) => {
  const lines = (values ?? []).filter((value): value is string =>
    Boolean(value)
  );

  if (!lines.length) {
    return EMPTY_VITAL_VALUE;
  }

  return (
    <div className={`${baseClass}__lines`}>
      {lines.map((line) => (
        <div key={line}>{line}</div>
      ))}
    </div>
  );
};

interface IVital {
  key: VitalKey;
  label: string;
  render: (host: IHost) => React.ReactNode;
  /** Multi-line values (JSON previews, line-per-entry lists) need to wrap
   * instead of truncating with an ellipsis (DataSet's default). */
  multiline?: boolean;
}

const VITALS: IVital[] = [
  {
    key: "accessibility_settings",
    label: "Accessibility settings",
    render: (host) => renderJson(host.accessibility_settings),
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
    render: (host) => renderJson(host.mdm_options),
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
    render: (host) => renderJson(host.organization_info),
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
  },
  {
    key: "service_subscriptions",
    label: "Service subscriptions",
    // TODO(nulmete): Per-subscription objects can carry many mostly-absent fields (see
    // MDMAppleServiceSubscription) — showing just the slot until product
    // confirms which other field(s) are worth surfacing here.
    render: (host) =>
      renderLines(host.service_subscriptions?.map((sub) => sub.slot)),
    multiline: true,
  },
  {
    key: "supplemental_build_version",
    label: "Supplemental build version",
    render: (host) => renderText(host.supplemental_build_version),
  },
  {
    key: "supplemental_os_version_extra",
    label: "Supplemental OS version extra",
    render: (host) => renderText(host.supplemental_os_version_extra),
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
VITALS.sort((a, b) => a.label.localeCompare(b.label));

interface IVitalsModal {
  host: IHost;
  onExit: () => void;
}

const VitalsModal = ({ host, onExit }: IVitalsModal) => {
  // enrollment_status is the full MdmEnrollmentStatus union; a non-iOS status
  // (or null) simply has no entry, leaving every vital supported.
  const unsupportedVitals =
    UNSUPPORTED_VITALS_BY_ENROLLMENT[
      host.mdm?.enrollment_status as IosOrIpadosEnrollmentStatus
    ];

  return (
    <Modal title="Vitals" className={baseClass} onExit={onExit} width="large">
      <>
        <dl className={`${baseClass}__vitals`}>
          {VITALS.map(({ key, label, render, multiline }) => {
            const isUnsupported = unsupportedVitals?.includes(key) ?? false;
            const value = isUnsupported ? (
              <TooltipWrapper tipContent={NOT_SUPPORTED_VITAL_TOOLTIP}>
                Not supported
              </TooltipWrapper>
            ) : (
              render(host)
            );

            return (
              <DataSet
                key={key}
                title={label}
                value={value}
                multiline={multiline && !isUnsupported}
              />
            );
          })}
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
