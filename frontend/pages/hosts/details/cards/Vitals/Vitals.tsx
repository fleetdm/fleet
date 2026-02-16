import React from "react";
import classnames from "classnames";

import { IAppleDeviceUpdates } from "interfaces/config";
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
  MDM_ENROLLMENT_STATUS_UI_MAP,
} from "interfaces/mdm";
import { ROLLING_ARCH_LINUX_VERSIONS } from "interfaces/software";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  MDM_STATUS_TOOLTIP,
  BATTERY_TOOLTIP,
} from "utilities/constants";
import {
  humanHostMemory,
  wrapFleetHelper,
  removeOSPrefix,
  compareVersions,
} from "utilities/helpers";

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
import { getCityCountryLocation } from "../../modals/LocationModal/LocationModal";

interface IVitalsProps {
  vitalsData: { [key: string]: any };
  munki?: IMunkiData | null;
  mdm?: IHostMdmData;
  osVersionRequirement?: IAppleDeviceUpdates;
  className?: string;
  toggleLocationModal?: () => void;
}

const baseClass = "vitals-card";

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
    platform === "arch" ||
    platform === "archarm" ||
    platform === "manjaro" ||
    platform === "manjaro-arm"
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

const Vitals = ({
  vitalsData,
  munki,
  mdm,
  osVersionRequirement,
  className,
  toggleLocationModal,
}: IVitalsProps) => {
  const isIosOrIpadosHost = isIPadOrIPhone(vitalsData.platform);
  const isAndroidHost = isAndroid(vitalsData.platform);
  const isChromeHost = isChrome(vitalsData.platform);

  // Generate the device ID data set based on MDM enrollment status. This is
  // either the Enrollment ID for personal (BYOD) devices or the Serial number
  // for business owned devices.
  const generateDeviceIdDataSet = () => {
    // we will default to showing the Serial number dataset. If the below consitions
    // evaludate to true, we will instead show the Enrollment ID dataset.
    let deviceIdDataSet = (
      <DataSet
        title="Serial number"
        value={<TooltipTruncatedText value={vitalsData.hardware_serial} />}
      />
    );

    // if the host is an Android host and it is not enrolled in MDM personally,
    // we do not show the device ID dataset at all.
    if (isAndroidHost && mdm && mdm.enrollment_status !== "On (personal)") {
      return null;
    }

    // for all host types, we show the Enrollment ID dataset if the host
    // is enrolled in MDM personally. Personal (BYOD) devices do not report
    // their serial numbers, so we show the Enrollment ID instead.
    if (mdm && isBYODAccountDrivenUserEnrollment(mdm.enrollment_status)) {
      deviceIdDataSet = (
        <DataSet
          title={
            <TooltipWrapper tipContent="Enrollment ID is a unique identifier for personal hosts. Personal (BYOD) devices don't report their serial numbers. The Enrollment ID changes with each enrollment.">
              Enrollment ID
            </TooltipWrapper>
          }
          value={<TooltipTruncatedText value={vitalsData.uuid} />}
        />
      );
    }
    return deviceIdDataSet;
  };

  const renderHardwareSerialAndIPs = () => {
    const DeviceIdDataSet = generateDeviceIdDataSet();

    // for an Android host, we show the either only the Hardware model or
    // Hardware model and Enrollment ID dataset.
    if (isAndroidHost) {
      return (
        <>
          {DeviceIdDataSet}
          <DataSet
            title="Hardware model"
            value={<TooltipTruncatedText value={vitalsData.hardware_model} />}
          />
        </>
      );
    }

    // for iOS and iPadOS hosts, we show to show a device ID dataset
    // (either Serial number or Enrollment ID) and the hardware model.
    if (isIosOrIpadosHost) {
      return (
        <>
          {DeviceIdDataSet}
          <DataSet
            title="Hardware model"
            value={<TooltipTruncatedText value={vitalsData.hardware_model} />}
          />
        </>
      );
    }

    // all other hosts will show the Hardware model, IP addresses, and a device ID dataset
    // (either Serial number or Enrollment ID).
    return (
      <>
        <DataSet
          title="Hardware model"
          value={<TooltipTruncatedText value={vitalsData.hardware_model} />}
        />
        {DeviceIdDataSet}
        <DataSet
          title="Private IP address"
          value={<TooltipTruncatedText value={vitalsData.primary_ip} />}
        />
        <DataSet
          title={
            <TooltipWrapper tipContent="The IP address the host uses to connect to Fleet.">
              Public IP address
            </TooltipWrapper>
          }
          value={<TooltipTruncatedText value={vitalsData.public_ip} />}
        />
      </>
    );
  };

  const renderMunkiData = () => {
    return munki ? (
      <>
        <DataSet
          title="Munki version"
          value={munki.version || DEFAULT_EMPTY_CELL_VALUE}
        />
      </>
    ) : null;
  };

  const renderMdmData = () => {
    if (!mdm?.enrollment_status) {
      return null;
    }
    return (
      <>
        <DataSet
          title="MDM status"
          value={
            <TooltipWrapper
              tipContent={MDM_STATUS_TOOLTIP[mdm.enrollment_status]}
              underline={mdm.enrollment_status !== "Off"}
            >
              {MDM_ENROLLMENT_STATUS_UI_MAP[mdm.enrollment_status].displayName}
            </TooltipWrapper>
          }
        />
        <DataSet
          title="MDM server URL"
          value={
            <TooltipTruncatedText
              value={mdm.server_url || DEFAULT_EMPTY_CELL_VALUE}
            />
          }
        />
      </>
    );
  };

  const renderTimezone = () => {
    if (!isIosOrIpadosHost || !vitalsData?.timezone) {
      return null;
    }
    return (
      <DataSet
        title="Timezone"
        value={
          <TooltipTruncatedText
            value={vitalsData.timezone || DEFAULT_EMPTY_CELL_VALUE}
          />
        }
      />
    );
  };

  const renderGeolocation = () => {
    const geolocation = vitalsData.geolocation;

    const isAdeIDevice =
      isIosOrIpadosHost && mdm?.enrollment_status === "On (automatic)";

    if (!isAdeIDevice && !geolocation) {
      return null;
    }

    const geoLocationButton = (
      <Button variant="text-link" onClick={toggleLocationModal}>
        {isAdeIDevice ? "Show location" : getCityCountryLocation(geolocation)}
      </Button>
    );
    return <DataSet title="Location" value={geoLocationButton} />;
  };

  const renderBattery = () => {
    if (
      vitalsData.batteries === null ||
      typeof vitalsData.batteries !== "object" ||
      vitalsData.batteries?.[0]?.health === "Unknown"
    ) {
      return null;
    }
    return (
      <DataSet
        title="Battery condition"
        value={
          <TooltipWrapper
            tipContent={BATTERY_TOOLTIP[vitalsData.batteries?.[0]?.health]}
          >
            {vitalsData.batteries?.[0]?.health}
          </TooltipWrapper>
        }
      />
    );
  };

  // TODO(android): confirm visible fields using actual android device data

  const {
    platform,
    os_version,
    disk_encryption_enabled: diskEncryptionEnabled,
  } = vitalsData;

  const renderDiskSpaceSummary = () => {
    // Hide disk space field if storage measurement is not supported (sentinel value -1)
    if (
      typeof vitalsData.gigs_disk_space_available === "number" &&
      vitalsData.gigs_disk_space_available < 0
    ) {
      return null;
    }

    const title = isAndroidHost ? (
      <TooltipWrapper tipContent="Includes internal and removable storage (e.g. microSD card).">
        Disk space available
      </TooltipWrapper>
    ) : (
      "Disk space available"
    );

    return (
      <DataSet
        title={title}
        value={
          <DiskSpaceIndicator
            gigsDiskSpaceAvailable={vitalsData.gigs_disk_space_available}
            percentDiskSpaceAvailable={vitalsData.percent_disk_space_available}
            gigsTotalDiskSpace={vitalsData.gigs_total_disk_space}
            gigsAllDiskSpace={vitalsData.gigs_all_disk_space}
            platform={platform}
            tooltipPosition="bottom"
          />
        }
      />
    );
  };

  const renderDiskEncryptionSummary = () => {
    if (!platformSupportsDiskEncryption(platform, os_version)) {
      return <></>;
    }
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

    return (
      <DataSet
        title="Disk encryption"
        value={
          <TooltipWrapper tipContent={tooltipMessage}>
            {statusText}
          </TooltipWrapper>
        }
      />
    );
  };

  const renderAgentSummary = () => {
    if (isIosOrIpadosHost || isAndroidHost) {
      return null;
    }

    const {
      orbit_version,
      osquery_version,
      fleet_desktop_version,
    } = vitalsData;

    if (isChromeHost) {
      return <DataSet title="Agent" value={osquery_version} />;
    }

    if (orbit_version !== DEFAULT_EMPTY_CELL_VALUE) {
      return (
        <DataSet
          title="Agent"
          value={
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
          }
        />
      );
    }
    return <DataSet title="Osquery" value={osquery_version} />;
  };

  const renderOperatingSystemSummary = () => {
    // No tooltip if minimum version is not set, including all Windows, Linux, ChromeOS, Android operating systems
    if (!osVersionRequirement?.minimum_version) {
      const version = vitalsData.os_version;
      const versionForRender = ROLLING_ARCH_LINUX_VERSIONS.includes(version) ? (
        // wrap a tooltip around the "rolling" suffix
        <>
          {version.slice(0, -8)}
          <TooltipWrapperArchLinuxRolling />
        </>
      ) : (
        <TooltipTruncatedText value={version} />
      );
      return (
        <DataSet
          title="Operating system"
          value={versionForRender}
          className={`${baseClass}__os-data-set`}
        />
      );
    }

    const osVersionWithoutPrefix = removeOSPrefix(vitalsData.os_version);
    const osVersionRequirementMet =
      compareVersions(
        osVersionWithoutPrefix,
        osVersionRequirement.minimum_version
      ) >= 0;

    return (
      <DataSet
        title="Operating system"
        value={
          <span className={`${baseClass}__os-version`}>
            {!osVersionRequirementMet && (
              <Icon name="error-outline" color="ui-fleet-black-75" />
            )}
            <TooltipWrapper
              className={`${baseClass}__os-version-tooltip`}
              tipContent={
                osVersionRequirementMet ? (
                  <>
                    {vitalsData.os_version}
                    <br />
                    Meets minimum version requirement.
                  </>
                ) : (
                  <>
                    {vitalsData.os_version}
                    <br />
                    Does not meet minimum version requirement.
                    <br />
                    Deadline to update: {osVersionRequirement.deadline}
                  </>
                )
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
    );
  };

  const renderVitalsAlphabetically = () => {
    const vitals: { sortKey: string; element: React.ReactNode }[] = [];

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

      if (isChromeHost) {
        vitals.push({
          sortKey: "Agent",
          element: (
            <DataSet key="agent" title="Agent" value={osquery_version} />
          ),
        });
      } else if (orbit_version !== DEFAULT_EMPTY_CELL_VALUE) {
        vitals.push({
          sortKey: "Agent",
          element: (
            <DataSet
              key="agent"
              title="Agent"
              value={
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
              }
            />
          ),
        });
      } else {
        vitals.push({
          sortKey: "Osquery",
          element: (
            <DataSet key="osquery" title="Osquery" value={osquery_version} />
          ),
        });
      }
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

    // Disk space available
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

    // Enrollment ID (for BYOD devices)
    const DeviceIdDataSet = generateDeviceIdDataSet();
    if (DeviceIdDataSet) {
      // Determine if it's Enrollment ID or Serial number for sorting
      const isBYOD =
        mdm && isBYODAccountDrivenUserEnrollment(mdm.enrollment_status);
      vitals.push({
        sortKey: isBYOD ? "Enrollment ID" : "Serial number",
        element: React.cloneElement(DeviceIdDataSet, {
          key: isBYOD ? "enrollment-id" : "serial-number",
        }),
      });
    }

    // Hardware model
    vitals.push({
      sortKey: "Hardware model",
      element: (
        <DataSet
          key="hardware-model"
          title="Hardware model"
          value={<TooltipTruncatedText value={vitalsData.hardware_model} />}
        />
      ),
    });

    // Last restarted
    if (!isIosOrIpadosHost && !isAndroidHost) {
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

    if (isAdeIDevice || geolocation) {
      const geoLocationButton = (
        <Button variant="text-link" onClick={toggleLocationModal}>
          {isAdeIDevice ? "Show location" : getCityCountryLocation(geolocation)}
        </Button>
      );
      vitals.push({
        sortKey: "Location",
        element: (
          <DataSet key="location" title="Location" value={geoLocationButton} />
        ),
      });
    }

    // MDM server URL
    if (mdm?.enrollment_status) {
      vitals.push({
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
      });
    }

    // MDM status
    if (mdm?.enrollment_status) {
      vitals.push({
        sortKey: "MDM status",
        element: (
          <DataSet
            key="mdm-status"
            title="MDM status"
            value={
              <TooltipWrapper
                tipContent={MDM_STATUS_TOOLTIP[mdm.enrollment_status]}
                underline={mdm.enrollment_status !== "Off"}
              >
                {
                  MDM_ENROLLMENT_STATUS_UI_MAP[mdm.enrollment_status]
                    .displayName
                }
              </TooltipWrapper>
            }
          />
        ),
      });
    }

    // Memory
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

    // Munki version
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
    if (!osVersionRequirement?.minimum_version) {
      const version = vitalsData.os_version;
      const versionForRender = ROLLING_ARCH_LINUX_VERSIONS.includes(version) ? (
        <>
          {version.slice(0, -8)}
          <TooltipWrapperArchLinuxRolling />
        </>
      ) : (
        <TooltipTruncatedText value={version} />
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
      const osVersionWithoutPrefix = removeOSPrefix(vitalsData.os_version);
      const osVersionRequirementMet =
        compareVersions(
          osVersionWithoutPrefix,
          osVersionRequirement.minimum_version
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
                    osVersionRequirementMet ? (
                      <>
                        {vitalsData.os_version}
                        <br />
                        Meets minimum version requirement.
                      </>
                    ) : (
                      <>
                        {vitalsData.os_version}
                        <br />
                        Does not meet minimum version requirement.
                        <br />
                        Deadline to update: {osVersionRequirement.deadline}
                      </>
                    )
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

    // Private IP address
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
    }

    // Processor type
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

    // Public IP address
    if (!isIosOrIpadosHost && !isAndroidHost) {
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
    }

    // Timezone
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

    // Sort alphabetically by title and render
    return (
      <>
        {vitals
          .sort((a, b) => a.sortKey.localeCompare(b.sortKey))
          .map((dataset) => dataset.element)}
      </>
    );
  };

  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <CardHeader header="Vitals" />
      <div className={`${baseClass}__info-grid`}>
        {renderVitalsAlphabetically()}
      </div>
    </Card>
  );
};

export default Vitals;
