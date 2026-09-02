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

/** AMAPI reports the manufacturer exactly as the device sends it, which is
 * often all lowercase ("samsung"). Only the first letter is touched, so a name
 * that capitalizes itself ("OnePlus", "LGE") keeps its own casing. */
const displayManufacturer = (value?: string) =>
  value
    ? displayText(value.charAt(0).toUpperCase() + value.slice(1))
    : DEFAULT_EMPTY_CELL_VALUE;

/** The trailing security patch level in an Android os_version, which the
 * server folds in — "Android 16 (2026-01-01)". */
const ANDROID_OS_PATCH_LEVEL = / \(\d{4}-\d{2}-\d{2}\)$/;

/** Drops the security patch level from an Android OS version for display. It
 * has its own vital (Security update version), so the parenthetical would just
 * repeat it. Only a value ending in a date is touched, so an unexpected format
 * is shown as sent. The API keeps returning the full string — this is display
 * only, and every other surface still shows it. */
export const stripAndroidOSPatchLevel = (osVersion?: string) =>
  osVersion ? osVersion.replace(ANDROID_OS_PATCH_LEVEL, "") : osVersion;

const SECURITY_PATCH_LEVEL_FORMAT = /^(\d{4})-(\d{2})-(\d{2})$/;

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
  const parts = SECURITY_PATCH_LEVEL_FORMAT.exec(value);
  if (!parts) {
    return value;
  }
  const [, year, month, day] = parts;
  const asDate = `${value}T00:00:00`;
  const parsed = new Date(asDate);
  // Being date-shaped isn't enough: the value can still name a day that
  // doesn't exist, and V8 rolls "2026-02-30" over to March 2 rather than
  // rejecting it, so the parsed components are compared back to the input.
  // An unparseable value yields NaN components, which fail this too.
  if (
    parsed.getFullYear() !== Number(year) ||
    parsed.getMonth() + 1 !== Number(month) ||
    parsed.getDate() !== Number(day)
  ) {
    return value;
  }
  return readableDate(asDate);
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
  /** The vital's display label. Sorts the card, and is kebab-cased into the
   * DataSet's React key. */
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
      sortKey: "Play Protect enabled",
      title: "Play Protect enabled",
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
      value: displayManufacturer(manufacturer),
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
  // Ownership has to be positively confirmed, so anything unknown resolves to
  // not showing these rather than to showing them: they're hardware
  // identifiers, and not showing one costs the admin far less than surfacing
  // it on a device that turns out to be personal. That covers the My device
  // page, which deliberately passes no `mdm` at all, as well as a payload
  // that omits is_personal_enrollment.
  const isCompanyOwned =
    mdm?.is_personal_enrollment === false &&
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
    // the two, so it only earns a row on the devices that report it. The
    // empty-cell string is checked for as well as a missing value, since
    // normalizeEmptyValues rewrites an empty one to it.
    if (meid && meid !== DEFAULT_EMPTY_CELL_VALUE) {
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
