import React from "react";

import {
  IHost,
  IHostMdmAndroidTelephonyInfo,
  IHostMdmData,
} from "interfaces/host";
import { wasBYODEnrolled } from "interfaces/mdm";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { readableDate } from "utilities/helpers";

import CustomLink from "components/CustomLink";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";

import type { VitalForSort } from "./Vitals";

/** Display labels for the AMAPI enums Fleet stores verbatim. Values Google
 * adds later fall through to the raw enum string rather than disappearing,
 * so a new member still shows the admin something. */
const ENCRYPTION_STATUS_LABELS: Record<string, string> = {
  UNSUPPORTED: "Unsupported",
  INACTIVE: "Inactive",
  ACTIVATING: "Activating",
  ACTIVE: "Active",
  ACTIVE_DEFAULT_KEY: "Active (default key)",
  ACTIVE_PER_USER: "Active (per user)",
};

const SECURITY_POSTURE_LABELS: Record<string, string> = {
  SECURE: "Secure",
  AT_RISK: "At risk",
  POTENTIALLY_COMPROMISED: "Potentially compromised",
};

/** AMAPI names the three pending states "…_AVAILABLE"; the UI says "pending". */
const SYSTEM_UPDATE_STATUS_LABELS: Record<string, string> = {
  UP_TO_DATE: "Up to date",
  UNKNOWN_UPDATE_AVAILABLE: "Update pending",
  SECURITY_UPDATE_AVAILABLE: "Security update pending",
  OS_UPDATE_AVAILABLE: "OS update pending",
};

const displayEnum = (
  value: string | undefined,
  labels: Record<string, string>
) => {
  if (!value) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }
  // Own-property check, not a bare lookup: the value is a device-reported
  // string stored verbatim, and "constructor" or "toString" would otherwise
  // resolve to an Object.prototype member instead of falling back.
  return Object.prototype.hasOwnProperty.call(labels, value)
    ? labels[value]
    : value;
};

const displayBoolean = (value?: boolean | null) => {
  if (value === undefined || value === null) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }
  return value ? "True" : "False";
};

const displayText = (value?: string | null) =>
  value ? <TooltipTruncatedText value={value} /> : DEFAULT_EMPTY_CELL_VALUE;

const SECURITY_PATCH_LEVEL_FORMAT = /^\d{4}-\d{2}-\d{2}$/;

/** AMAPI reports the security patch level as a YYYY-MM-DD date, which is shown
 * as a readable one. The time is appended so it parses in the viewer's own
 * timezone — `new Date("2026-05-01")` is UTC midnight, which formats as the
 * previous day anywhere west of UTC.
 *
 * The value is device-reported and stored verbatim, so it is shown as sent
 * unless it parses to a real date. Date-shaped but impossible values
 * ("2026-13-45") matter: readableDate would hand an Invalid Date to
 * Intl.DateTimeFormat.format, which throws and takes down the whole card. */
const displaySecurityUpdateVersion = (value?: string) => {
  if (!value) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }
  if (!SECURITY_PATCH_LEVEL_FORMAT.test(value)) {
    return value;
  }
  const asDate = `${value}T00:00:00`;
  return Number.isNaN(new Date(asDate).getTime())
    ? value
    : readableDate(asDate);
};

/** A vital title whose tooltip explains the vital and links out to the docs. */
const titleWithLearnMore = (
  title: string,
  description: string,
  learnMoreSlug: string
) => (
  <TooltipWrapper
    tipContent={
      <>
        {description}{" "}
        <CustomLink
          text="Learn more"
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/${learnMoreSlug}`}
          newTab
          variant="tooltip-link"
        />
      </>
    }
  >
    {title}
  </TooltipWrapper>
);

/** Renders a per-SIM value. A dual-SIM device reports one telephonyInfo per
 * card, so the first is shown with a tooltip listing all of them.
 *
 * Distinct values only: both SIMs of a dual-SIM device are usually on the same
 * carrier, and a tooltip reading "Verizon / Verizon" tells the user nothing.
 * Deduplicating also keeps the value usable as a React key. */
const displayTelephonyValue = (
  telephonyInfos: IHostMdmAndroidTelephonyInfo[] | undefined,
  getValue: (info: IHostMdmAndroidTelephonyInfo) => string | undefined
) => {
  // normalizeEmptyValues rewrites an empty array to the empty-cell string
  // before the card ever sees it, so this can't assume an array.
  const infos = Array.isArray(telephonyInfos) ? telephonyInfos : [];
  const values = Array.from(
    new Set(infos.map(getValue).filter((value): value is string => !!value))
  );

  if (!values.length) {
    return DEFAULT_EMPTY_CELL_VALUE;
  }

  if (values.length === 1) {
    return <TooltipTruncatedText value={values[0]} />;
  }

  return (
    <TooltipTruncatedText
      value={values[0]}
      alwaysShowTooltip
      tooltip={
        // Left-align to override the tooltip's default centered text.
        <div style={{ textAlign: "left" }}>
          {values.map((value) => (
            <div key={value}>{value}</div>
          ))}
        </div>
      }
    />
  );
};

interface IAndroidVital {
  /** Also the DataSet key, kebab-cased. */
  sortKey: string;
  title: React.ReactNode;
  value: React.ReactNode;
}

/** Builds the Android-only host vitals. Every row renders for an Android host
 * whether or not the device reported it — AMAPI populates each section only
 * when the applied policy enables the matching status reporting setting, so an
 * unreported vital shows as empty rather than vanishing. */
const buildAndroidHostVitals = (
  vitalsData: Partial<IHost>,
  mdm?: IHostMdmData
): VitalForSort[] => {
  const {
    adb_enabled: adbEnabled,
    passcode_protected: passcodeProtected,
    play_protect_enabled: playProtectEnabled,
    encryption_type: encryptionType,
    manufacturer,
    security_update_version: securityUpdateVersion,
    device_kernel_version: deviceKernelVersion,
    bootloader_version: bootloaderVersion,
    system_update_status: systemUpdateStatus,
    security_posture: securityPosture,
    imei,
    meid,
    telephony_infos: telephonyInfos,
  } = vitalsData;

  const rows: IAndroidVital[] = [
    {
      sortKey: "Bootloader version",
      title: "Bootloader version",
      value: displayText(bootloaderVersion),
    },
    {
      sortKey: "Encryption status",
      title: titleWithLearnMore(
        "Encryption status",
        "Shows whether this device's storage is encrypted.",
        "android-encryption-status"
      ),
      value: displayEnum(encryptionType, ENCRYPTION_STATUS_LABELS),
    },
    {
      sortKey: "Google Play Protect enabled",
      title: "Google Play Protect enabled",
      value: displayBoolean(playProtectEnabled),
    },
    {
      sortKey: "Kernel version",
      title: "Kernel version",
      value: displayText(deviceKernelVersion),
    },
    {
      sortKey: "Manufacturer",
      title: "Manufacturer",
      value: displayText(manufacturer),
    },
    {
      sortKey: "Passcode set",
      title: "Passcode set",
      value: displayBoolean(passcodeProtected),
    },
    {
      sortKey: "Security posture",
      title: titleWithLearnMore(
        "Security posture",
        "Google's risk assessment for this host, based on signals like screen lock, disk encryption, and installed apps.",
        "security-posture"
      ),
      value: displayEnum(securityPosture, SECURITY_POSTURE_LABELS),
    },
    {
      sortKey: "Security update version",
      title: "Security update version",
      value: displaySecurityUpdateVersion(securityUpdateVersion),
    },
    {
      sortKey: "Software update status",
      title: titleWithLearnMore(
        "Software update status",
        "Shows whether an Android OS update is available for this host.",
        "software-update-status"
      ),
      value: displayEnum(systemUpdateStatus, SYSTEM_UPDATE_STATUS_LABELS),
    },
    {
      sortKey: "USB debugging enabled",
      title: "USB debugging enabled",
      value: displayBoolean(adbEnabled),
    },
  ];

  // AMAPI reports telephony info, IMEI and MEID only for company-owned
  // devices, and the server withholds all three for a personal enrollment — so
  // on a BYOD host these rows could only ever be empty. BYOD is keyed off the
  // same predicate as the Enrollment ID row, which holds after the host
  // unenrolls.
  //
  // Absent MDM data means BYOD can't be ruled out, so the rows stay hidden.
  // That's the My device page, which deliberately passes no `mdm` — and these
  // are hardware identifiers, so "unknown ownership" resolves to not showing
  // them rather than to showing them.
  const isCompanyOwned =
    !!mdm &&
    !wasBYODEnrolled(mdm.enrollment_status, mdm.is_personal_enrollment);

  if (isCompanyOwned) {
    rows.push(
      {
        sortKey: "Carrier",
        title: "Carrier",
        value: displayTelephonyValue(
          telephonyInfos,
          (info) => info.carrier_name
        ),
      },
      {
        sortKey: "IMEI",
        title: "IMEI",
        value: displayText(imei),
      },
      {
        sortKey: "Phone number",
        title: "Phone number",
        value: displayTelephonyValue(
          telephonyInfos,
          (info) => info.phone_number
        ),
      }
    );

    // MEID is the CDMA counterpart of IMEI and a device reports at most one of
    // the two, so it only earns a row on the devices that report it.
    if (meid) {
      rows.push({ sortKey: "MEID", title: "MEID", value: displayText(meid) });
    }
  }

  return rows.map(({ sortKey, title, value }) => ({
    sortKey,
    element: (
      <DataSet
        key={sortKey.toLowerCase().replace(/\s+/g, "-")}
        title={title}
        value={value}
      />
    ),
  }));
};

export default buildAndroidHostVitals;
