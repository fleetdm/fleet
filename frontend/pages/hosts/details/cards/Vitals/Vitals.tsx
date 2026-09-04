import React, { useEffect, useRef, useState } from "react";
import classnames from "classnames";

import { IHostCustomVital } from "interfaces/custom_host_vitals";
import { IHostMdmData, IMunkiData } from "interfaces/host";
import {
  isAndroid,
  isIPadOrIPhone,
  isChrome,
  platformSupportsDiskEncryption,
  DiskEncryptionSupportedPlatform,
} from "interfaces/platform";
import {
  isBYODAccountDrivenUserEnrollment,
  wasBYODEnrolled,
  MDM_ENROLLMENT_STATUS_UI_MAP,
} from "interfaces/mdm";
import { ROLLING_ARCH_LINUX_VERSIONS } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE, BATTERY_TOOLTIP } from "utilities/constants";
import {
  humanHostMemory,
  wrapFleetHelper,
  removeOSPrefix,
  compareVersions,
} from "utilities/helpers";
import { getHardwareModelDisplay } from "pages/hosts/helpers";

import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Card from "components/Card";
import DataSet from "components/DataSet";
import CardHeader from "components/CardHeader";
import TooltipWrapperArchLinuxRolling from "components/TooltipWrapperArchLinuxRolling";
import Icon from "components/Icon/Icon";
import Button from "components/buttons/Button";

import DiskSpaceIndicator from "pages/hosts/components/DiskSpaceIndicator";
import buildAndroidHostVitals, {
  stripAndroidOSPatchLevel,
} from "./androidVitals";
import { getCityCountryLocation } from "../../modals/LocationModal/LocationModal";

/** Everything buildHostVitals needs to render the pre-existing host vitals.
 * Shared with the "View all" modal so both surfaces build the same rows from
 * one implementation. */
export interface IHostVitalsSources {
  vitalsData: { [key: string]: any };
  munki?: IMunkiData | null;
  mdm?: IHostMdmData;
  /** The OS version the host is required to reach, resolved per host by the
   * server. "Pending" while Fleet is still working it out for a "latest"
   * requirement, and undefined when OS updates aren't configured. */
  osUpdateMinimumVersion?: string | null;
  /** The deadline for osUpdateMinimumVersion, following the same states. */
  osUpdateDeadline?: string | null;
  /**
   * Opens the Location modal. Presence of this handler also makes the
   * Location row interactive — omit it for read-only contexts (e.g., the
   * My device page) so the row renders as plain text instead of a link.
   */
  toggleLocationModal?: () => void;
  /**
   * Opens the MDM status modal. Presence of this handler also makes the
   * MDM status row interactive — omit it for read-only contexts (e.g.,
   * the My device page) so the row renders as plain text instead of a link.
   */
  toggleMDMStatusModal?: () => void;
  customHostVitals?: IHostCustomVital[];
  onEditCustomHostVital?: (vital: IHostCustomVital) => void;
}

interface IVitalsProps extends IHostVitalsSources {
  className?: string;
  /**
   * Opens the "View all" vitals modal (iOS/iPadOS only). Presence of this
   * handler also makes the header button appear — omit it for read-only
   * contexts (e.g., the My device page) so the button doesn't render at all.
   */
  toggleVitalsModal?: () => void;
}

export type VitalForSort = { sortKey: string; element: React.ReactNode };

/** How many grid rows an iOS/iPadOS card fills before the rest move behind "View all" (#39281). */
const IOS_CARD_VITAL_ROWS = 2;

/** Column count assumed until the grid can be measured — matches the widest
 * breakpoint in _styles.scss, so the first paint errs toward showing too many
 * vitals rather than hiding ones that fit. */
const FALLBACK_COLUMN_COUNT = 6;

/** Reads the grid's resolved column count. Browsers report
 * grid-template-columns as a resolved track list ("214px 214px …"), so the
 * track count is the column count; environments without layout (jsdom) report
 * the unresolved value instead, hence the fallback. */
const getGridColumnCount = (grid: HTMLElement | null) => {
  if (!grid) return FALLBACK_COLUMN_COUNT;

  const tracks = window
    .getComputedStyle(grid)
    .gridTemplateColumns.split(" ")
    .filter((track) => track.endsWith("px"));

  return tracks.length || FALLBACK_COLUMN_COUNT;
};

const baseClass = "vitals-card";

/** What the API sends for a host's OS update target while Fleet is still
 * resolving a "latest" requirement. Shown to the user as-is, per the design. */
const OS_UPDATE_REQUIREMENT_PENDING = "Pending";

const DISK_ENCRYPTION_MESSAGES = {
  darwin: {
    enabled: (
      <>
        The disk is encrypted. The user must enter their
        <br /> password when they start their computer.
      </>
    ),
    disabled: (
      <>
        The disk might be encrypted, but FileVault is off. The
        <br /> disk can be accessed without entering a password.
      </>
    ),
  },
  windows: {
    enabled: (
      <>
        The disk is encrypted. If recently turned on,
        <br /> encryption could take awhile.
      </>
    ),
    disabled: "The disk is unencrypted.",
  },
  linux: {
    enabled: "The disk is encrypted.",
    unknown: "The disk may be encrypted.",
  },
};

const getHostDiskEncryptionTooltipMessage = (
  platform: DiskEncryptionSupportedPlatform, // TODO: improve this type
  diskEncryptionEnabled = false
) => {
  if (platform === "chrome") {
    return "Fleet does not check for disk encryption on Chromebooks, as they are encrypted by default.";
  }

  if (
    platform === "rhel" ||
    platform === "ubuntu" ||
    platform === "zorin" ||
    platform === "arch" ||
    platform === "archarm" ||
    platform === "manjaro" ||
    platform === "manjaro-arm" ||
    platform === "cachyos" ||
    platform === "omarchy"
  ) {
    return DISK_ENCRYPTION_MESSAGES.linux[
      diskEncryptionEnabled ? "enabled" : "unknown"
    ];
  }

  // mac or windows
  return DISK_ENCRYPTION_MESSAGES[platform][
    diskEncryptionEnabled ? "enabled" : "disabled"
  ];
};

/** Builds the pre-existing host vitals as an unsorted list, so callers can
 * filter (the card, for iOS/iPadOS) or extend (the "View all" modal, which
 * appends the iOS/iPadOS-only vitals) before sorting. */
export const buildHostVitals = ({
  vitalsData,
  munki,
  mdm,
  osUpdateMinimumVersion,
  osUpdateDeadline,
  toggleLocationModal,
  toggleMDMStatusModal,
  customHostVitals,
  onEditCustomHostVital,
}: IHostVitalsSources): VitalForSort[] => {
  const isIosOrIpadosHost = isIPadOrIPhone(vitalsData.platform);
  const isAndroidHost = isAndroid(vitalsData.platform);
  const isChromeHost = isChrome(vitalsData.platform);

  const {
    platform,
    os_version,
    mdm_enrollment_hardware_attested,
    disk_encryption_enabled: diskEncryptionEnabled,
  } = vitalsData;

  const vitals: VitalForSort[] = [];

  vitals.push({
    sortKey: "Added to Fleet",
    element: (
      <DataSet
        key="added-to-fleet"
        title="Added to Fleet"
        value={
          <HumanTimeDiffWithFleetLaunchCutoff
            timeString={vitalsData.last_enrolled_at ?? "Unavailable"}
          />
        }
      />
    ),
  });

  // Agent / Osquery
  if (!isIosOrIpadosHost && !isAndroidHost) {
    const {
      orbit_version,
      osquery_version,
      fleet_desktop_version,
    } = vitalsData;

    const isChromeOrVanillaOsqueryHost =
      isChromeHost || orbit_version === DEFAULT_EMPTY_CELL_VALUE;

    vitals.push({
      sortKey: "Agent",
      element: (
        <DataSet
          key="agent"
          title="Agent"
          value={
            isChromeOrVanillaOsqueryHost ? (
              osquery_version
            ) : (
              <TooltipWrapper
                tipContent={
                  <>
                    osquery: {osquery_version}
                    <br />
                    Orbit: {orbit_version}
                    {fleet_desktop_version !== DEFAULT_EMPTY_CELL_VALUE && (
                      <>
                        <br />
                        Fleet Desktop: {fleet_desktop_version}
                      </>
                    )}
                  </>
                }
              >
                {orbit_version}
              </TooltipWrapper>
            )
          }
        />
      ),
    });
  }

  // Battery condition
  if (
    vitalsData.batteries !== null &&
    typeof vitalsData.batteries === "object" &&
    vitalsData.batteries?.[0]?.health !== "Unknown"
  ) {
    vitals.push({
      sortKey: "Battery condition",
      element: (
        <DataSet
          key="battery-condition"
          title="Battery condition"
          value={
            <TooltipWrapper
              tipContent={BATTERY_TOOLTIP[vitalsData.batteries?.[0]?.health]}
            >
              {vitalsData.batteries?.[0]?.health}
            </TooltipWrapper>
          }
        />
      ),
    });
  }

  // Disk encryption
  if (platformSupportsDiskEncryption(platform, os_version)) {
    const tooltipMessage = getHostDiskEncryptionTooltipMessage(
      platform,
      diskEncryptionEnabled
    );

    let statusText;
    switch (true) {
      case isChromeHost:
        statusText = "Always on";
        break;
      case diskEncryptionEnabled === true:
        statusText = "On";
        break;
      case diskEncryptionEnabled === false:
        statusText = "Off";
        break;
      case (diskEncryptionEnabled === null ||
        diskEncryptionEnabled === undefined) &&
        platformSupportsDiskEncryption(platform, os_version):
        statusText = "Unknown";
        break;
      default:
        // something unexpected happened on the way to this component, display whatever we got or
        // "Unknown" to draw attention to the issue.
        statusText = diskEncryptionEnabled || "Unknown";
    }

    vitals.push({
      sortKey: "Disk encryption",
      element: (
        <DataSet
          key="disk-encryption"
          title="Disk encryption"
          value={
            <TooltipWrapper tipContent={tooltipMessage}>
              {statusText}
            </TooltipWrapper>
          }
        />
      ),
    });
  }

  // Disk space
  if (
    !isChromeHost &&
    !(
      typeof vitalsData.gigs_disk_space_available === "number" &&
      vitalsData.gigs_disk_space_available < 0
    )
  ) {
    const title = isAndroidHost ? (
      <TooltipWrapper tipContent="Includes internal and removable storage (e.g. microSD card).">
        Disk space available
      </TooltipWrapper>
    ) : (
      "Disk space available"
    );

    vitals.push({
      sortKey: "Disk space available",
      element: (
        <DataSet
          key="disk-space-available"
          title={title}
          value={
            <DiskSpaceIndicator
              gigsDiskSpaceAvailable={vitalsData.gigs_disk_space_available}
              percentDiskSpaceAvailable={
                vitalsData.percent_disk_space_available
              }
              gigsTotalDiskSpace={vitalsData.gigs_total_disk_space}
              gigsAllDiskSpace={vitalsData.gigs_all_disk_space}
              platform={platform}
              tooltipPosition="bottom"
            />
          }
        />
      ),
    });
  }

  // Device identity
  if (
    mdm &&
    wasBYODEnrolled(mdm.enrollment_status, mdm.is_personal_enrollment)
  ) {
    //  Personal (BYOD) devices do not report their serial numbers, so show the enrollment id
    //  instead. This holds after the host unenrolls too, hence wasBYODEnrolled rather than the
    //  current enrollment status.
    vitals.push({
      sortKey: "Enrollment ID",
      element: (
        <DataSet
          key="enrollment-id"
          title={
            <TooltipWrapper tipContent="Enrollment ID is a unique identifier for personal hosts. Personal (BYOD) devices don't report their serial numbers. For personal (BYOD) Android hosts, this enrollment ID is stable across unenroll/re-enroll cycles.">
              Enrollment ID
            </TooltipWrapper>
          }
          value={<TooltipTruncatedText value={vitalsData.uuid} />}
        />
      ),
    });
  } else {
    // for all other host types, show the serial number
    vitals.push({
      sortKey: "Serial number",
      element: (
        <DataSet
          key="serial-number"
          title="Serial number"
          value={<TooltipTruncatedText value={vitalsData.hardware_serial} />}
        />
      ),
    });
  }

  // Hardware model
  const hardwareModelDisplay = getHardwareModelDisplay(
    vitalsData.platform,
    vitalsData.hardware_model,
    vitalsData.hardware_marketing_name
  );
  vitals.push({
    sortKey: "Hardware model",
    element: (
      <DataSet
        key="hardware-model"
        title="Hardware model"
        value={
          <TooltipTruncatedText
            value={hardwareModelDisplay.value}
            tooltip={hardwareModelDisplay.tooltip}
            alwaysShowTooltip={hardwareModelDisplay.alwaysShowTooltip}
          />
        }
      />
    ),
  });

  // Last restarted
  if (!isIosOrIpadosHost && !isAndroidHost && !isChromeHost) {
    vitals.push({
      sortKey: "Last restarted",
      element: (
        <DataSet
          key="last-restarted"
          title="Last restarted"
          value={
            <HumanTimeDiffWithFleetLaunchCutoff
              timeString={vitalsData.last_restarted_at}
            />
          }
        />
      ),
    });
  }

  // Location
  const geolocation = vitalsData.geolocation;
  const isAdeIDevice =
    isIosOrIpadosHost && mdm?.enrollment_status === "On (automatic)";

  if (isAdeIDevice ? toggleLocationModal : geolocation) {
    const label = isAdeIDevice
      ? "Show location"
      : getCityCountryLocation(geolocation);
    const locationValue = toggleLocationModal ? (
      <Button variant="link" onClick={toggleLocationModal}>
        {label}
      </Button>
    ) : (
      label
    );
    vitals.push({
      sortKey: "Location",
      element: (
        <DataSet
          className={`${baseClass}__location`}
          key="location"
          title="Location"
          value={locationValue}
        />
      ),
    });
  }

  // MDM attestation
  if (mdm_enrollment_hardware_attested) {
    vitals.push({
      sortKey: "MDM attestation",
      element: (
        <DataSet
          key="mdm-attestation"
          title="MDM attestation"
          value={
            <TooltipWrapper tipContent="Host provided a Managed Device Attestation signed by Apple at enrollment.">
              Yes
            </TooltipWrapper>
          }
        />
      ),
    });
  }

  // MDM
  if (mdm?.enrollment_status) {
    const mdmStatusLabel =
      MDM_ENROLLMENT_STATUS_UI_MAP[mdm.enrollment_status].displayName;
    vitals.push(
      {
        sortKey: "MDM status",
        element: (
          <DataSet
            key="mdm-status"
            title="MDM status"
            className={`${baseClass}__mdm-status`}
            value={
              <>
                {mdm.dep_profile_error && <Icon name="error" />}
                {toggleMDMStatusModal ? (
                  <Button variant="link" onClick={toggleMDMStatusModal}>
                    {mdmStatusLabel}
                  </Button>
                ) : (
                  mdmStatusLabel
                )}
              </>
            }
          />
        ),
      },

      {
        sortKey: "MDM server URL",
        element: (
          <DataSet
            key="mdm-server-url"
            title="MDM server URL"
            value={
              <TooltipTruncatedText
                value={mdm.server_url || DEFAULT_EMPTY_CELL_VALUE}
              />
            }
          />
        ),
      }
    );
  }

  if (!isIosOrIpadosHost) {
    vitals.push({
      sortKey: "Memory",
      element: (
        <DataSet
          key="memory"
          title="Memory"
          value={wrapFleetHelper(humanHostMemory, vitalsData.memory)}
        />
      ),
    });
  }

  if (munki) {
    vitals.push({
      sortKey: "Munki version",
      element: (
        <DataSet
          key="munki-version"
          title="Munki version"
          value={munki.version || DEFAULT_EMPTY_CELL_VALUE}
        />
      ),
    });
  }

  // Operating system
  // No tooltip if there's no requirement, including all Windows, Linux, ChromeOS, Android operating systems
  //
  // Checked as a number rather than for truthiness: the card's data is run
  // through normalizeEmptyValues, which rewrites an empty value to the
  // empty-cell string, and that string is truthy.
  const hasApiLevel = isAndroidHost && typeof vitalsData.api_level === "number";

  if (!osUpdateMinimumVersion) {
    const version = isAndroidHost
      ? stripAndroidOSPatchLevel(vitalsData.os_version)
      : vitalsData.os_version;
    const versionForRender = ROLLING_ARCH_LINUX_VERSIONS.includes(version) ? (
      <>
        {version.slice(0, -8)}&nbsp;
        <TooltipWrapperArchLinuxRolling />
      </>
    ) : (
      <TooltipTruncatedText
        value={version}
        // Android reports the API level separately from the OS version string;
        // it rides along on this row rather than getting a vital of its own.
        // The version is repeated in the tooltip because supplying `tooltip`
        // replaces the truncated-text tooltip, which would otherwise leave a
        // long version unreadable in a narrow column.
        tooltip={
          hasApiLevel ? (
            <>
              {version}
              <br />
              Android API level: {vitalsData.api_level}
            </>
          ) : undefined
        }
        alwaysShowTooltip={hasApiLevel}
      />
    );
    vitals.push({
      sortKey: "Operating system",
      element: (
        <DataSet
          key="operating-system"
          title="Operating system"
          value={versionForRender}
          className={`${baseClass}__os-data-set`}
        />
      ),
    });
  } else {
    // A "latest" requirement is resolved per host by the server, which sends
    // "Pending" until it has worked the target out. There's nothing to
    // compare against until then, so no compliance icon is shown.
    const isPendingRequirement =
      osUpdateMinimumVersion === OS_UPDATE_REQUIREMENT_PENDING;
    const osVersionRequirementMet =
      isPendingRequirement ||
      compareVersions(
        removeOSPrefix(vitalsData.os_version),
        osUpdateMinimumVersion
      ) >= 0;

    vitals.push({
      sortKey: "Operating system",
      element: (
        <DataSet
          key="operating-system"
          title="Operating system"
          value={
            <span className={`${baseClass}__os-version`}>
              {!osVersionRequirementMet && (
                <Icon name="error-outline" color="ui-fleet-black-75" />
              )}
              <TooltipWrapper
                className={`${baseClass}__os-version-tooltip`}
                tipContent={
                  <>
                    Minimum version required: <b>{osUpdateMinimumVersion}</b>
                    <br />
                    Deadline: <b>{osUpdateDeadline}</b>
                  </>
                }
              >
                <span className={`${baseClass}__os-version-text`}>
                  {vitalsData.os_version}
                </span>
              </TooltipWrapper>
            </span>
          }
          className={`${baseClass}__os-data-set`}
        />
      ),
    });
  }

  // IP addresses
  if (!isIosOrIpadosHost && !isAndroidHost) {
    vitals.push({
      sortKey: "Private IP address",
      element: (
        <DataSet
          key="private-ip-address"
          title="Private IP address"
          value={<TooltipTruncatedText value={vitalsData.primary_ip} />}
        />
      ),
    });
    vitals.push({
      sortKey: "Public IP address",
      element: (
        <DataSet
          key="public-ip-address"
          title={
            <TooltipWrapper tipContent="The IP address the host uses to connect to Fleet.">
              Public IP address
            </TooltipWrapper>
          }
          value={<TooltipTruncatedText value={vitalsData.public_ip} />}
        />
      ),
    });
    vitals.push({
      sortKey: "MAC address",
      element: (
        <DataSet
          key="mac-address"
          title="MAC address"
          value={<TooltipTruncatedText value={vitalsData.primary_mac} />}
        />
      ),
    });
  }

  if (!isIosOrIpadosHost) {
    vitals.push({
      sortKey: "Processor type",
      element: (
        <DataSet
          key="processor-type"
          title="Processor type"
          value={vitalsData.cpu_type}
        />
      ),
    });
  }

  if (isIosOrIpadosHost && vitalsData?.timezone) {
    vitals.push({
      sortKey: "Timezone",
      element: (
        <DataSet
          key="timezone"
          title="Timezone"
          value={
            <TooltipTruncatedText
              value={vitalsData.timezone || DEFAULT_EMPTY_CELL_VALUE}
            />
          }
        />
      ),
    });
  }

  if (isAndroidHost) {
    vitals.push(...buildAndroidHostVitals(vitalsData, mdm));
  }

  customHostVitals?.forEach((vital) => {
    const displayValue =
      vital.value === "" ? DEFAULT_EMPTY_CELL_VALUE : vital.value;
    const title = onEditCustomHostVital ? (
      <span className={`${baseClass}__custom-vital-title`}>
        {vital.name}
        <Button
          variant="subdued"
          size="small"
          onClick={() => onEditCustomHostVital(vital)}
          ariaLabel={`Edit ${vital.name}`}
          icon="pencil"
        />
      </span>
    ) : (
      vital.name
    );

    vitals.push({
      sortKey: vital.name,
      element: (
        <DataSet
          className={`${baseClass}__custom-vital`}
          key={`custom-host-vital-${vital.custom_host_vital_id}`}
          title={title}
          value={displayValue}
        />
      ),
    });
  });

  return vitals;
};

/** Sorts vitals by their display title. Exported so the "View all" modal
 * orders the combined list the same way the card does. */
export const sortHostVitals = (vitals: VitalForSort[]): VitalForSort[] =>
  [...vitals].sort((a, b) => a.sortKey.localeCompare(b.sortKey));

const Vitals = ({
  vitalsData,
  munki,
  mdm,
  osUpdateMinimumVersion,
  osUpdateDeadline,
  className,
  toggleLocationModal,
  toggleMDMStatusModal,
  toggleVitalsModal,
  customHostVitals,
  onEditCustomHostVital,
}: IVitalsProps) => {
  const isIosOrIpadosHost = isIPadOrIPhone(vitalsData.platform);
  // Unlike the Enrollment ID row, this keys off the *current* enrollment status on
  // purpose: the cap exists because a personally-enrolled device reports few vitals
  // right now, not because of how it was once enrolled.
  const showExpandedVitals =
    isIosOrIpadosHost &&
    !isBYODAccountDrivenUserEnrollment(mdm?.enrollment_status ?? null);

  const gridRef = useRef<HTMLDivElement>(null);
  const [columnCount, setColumnCount] = useState(FALLBACK_COLUMN_COUNT);

  useEffect(() => {
    const measure = () => setColumnCount(getGridColumnCount(gridRef.current));

    measure();

    if (!gridRef.current) return undefined;
    const observer = new ResizeObserver(measure);
    observer.observe(gridRef.current);
    return () => observer.disconnect();
  }, []);

  const allVitals = buildHostVitals({
    vitalsData,
    munki,
    mdm,
    osUpdateMinimumVersion,
    osUpdateDeadline,
    toggleLocationModal,
    toggleMDMStatusModal,
    customHostVitals,
    onEditCustomHostVital,
  });

  const sortedVitals = sortHostVitals(allVitals);

  // Only cap where "View all" can reach what's hidden. Capping without it
  // would drop vitals with no way to see them.
  const cardVitals =
    showExpandedVitals && toggleVitalsModal
      ? sortedVitals.slice(0, columnCount * IOS_CARD_VITAL_ROWS)
      : sortedVitals;

  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <div className={`${baseClass}__header`}>
        <CardHeader header="Vitals" />
        {showExpandedVitals && toggleVitalsModal && (
          <Button variant="subdued" size="small" onClick={toggleVitalsModal}>
            View all
          </Button>
        )}
      </div>
      <div className={`${baseClass}__info-grid`} ref={gridRef}>
        {cardVitals.map((vital) => vital.element)}
      </div>
    </Card>
  );
};

export default Vitals;
