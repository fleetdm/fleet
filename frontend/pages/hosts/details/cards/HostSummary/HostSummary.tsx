import React from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";
import { formatInTimeZone } from "date-fns-tz";
import {
  IHostMdmProfile,
  BootstrapPackageStatus,
  isWindowsDiskEncryptionStatus,
  isLinuxDiskEncryptionStatus,
} from "interfaces/mdm";
import { IOSSettings, IHostMaintenanceWindow } from "interfaces/host";
import { IAppleDeviceUpdates } from "interfaces/config";
import {
  DiskEncryptionSupportedPlatform,
  isDiskEncryptionSupportedLinuxPlatform,
  isOsSettingsDisplayPlatform,
  platformSupportsDiskEncryption,
} from "interfaces/platform";

import getHostStatusTooltipText from "pages/hosts/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import Card from "components/Card";
import DataSet from "components/DataSet";
import StatusIndicator from "components/StatusIndicator";
import IssuesIndicator from "pages/hosts/components/IssuesIndicator";
import DiskSpaceIndicator from "pages/hosts/components/DiskSpaceIndicator";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import {
  humanHostMemory,
  wrapFleetHelper,
  removeOSPrefix,
  compareVersions,
} from "utilities/helpers";
import {
  DATE_FNS_FORMAT_STRINGS,
  DEFAULT_EMPTY_CELL_VALUE,
} from "utilities/constants";
import { COLORS } from "styles/var/colors";

import OSSettingsIndicator from "./OSSettingsIndicator";
import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";

import {
  HostMdmDeviceStatusUIState,
  generateLinuxDiskEncryptionSetting,
  generateWinDiskEncryptionSetting,
} from "../../helpers";
import { DEVICE_STATUS_TAGS, REFETCH_TOOLTIP_MESSAGES } from "./helpers";

const baseClass = "host-summary";

interface IRefetchButtonProps {
  isDisabled: boolean;
  isFetching: boolean;
  tooltip?: React.ReactNode;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
}

const RefetchButton = ({
  isDisabled,
  isFetching,
  tooltip,
  onRefetchHost,
}: IRefetchButtonProps) => {
  const classNames = classnames({
    tooltip: isDisabled,
    "refetch-spinner": isFetching,
    "refetch-btn": !isFetching,
  });

  const buttonText = isFetching
    ? "Fetching fresh vitals...this may take a moment"
    : "Refetch";

  // add additonal props when we need to display a tooltip for the button
  const conditionalProps: { "data-tip"?: boolean; "data-for"?: string } = {};

  if (tooltip) {
    conditionalProps["data-tip"] = true;
    conditionalProps["data-for"] = "refetch-tooltip";
  }

  const renderTooltip = () => {
    return (
      <ReactTooltip
        place="top"
        effect="solid"
        id="refetch-tooltip"
        backgroundColor={COLORS["tooltip-bg"]}
      >
        <span className={`${baseClass}__tooltip-text`}>{tooltip}</span>
      </ReactTooltip>
    );
  };

  return (
    <>
      <div className={`${baseClass}__refetch`} {...conditionalProps}>
        <Button
          className={classNames}
          disabled={isDisabled}
          onClick={onRefetchHost}
          variant="text-icon"
        >
          <Icon name="refresh" color="core-fleet-blue" size="small" />
          {buttonText}
        </Button>
        {tooltip && renderTooltip()}
      </div>
    </>
  );
};

interface IBootstrapPackageData {
  status?: BootstrapPackageStatus | "";
  details?: string;
}

interface IHostSummaryProps {
  summaryData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  bootstrapPackageData?: IBootstrapPackageData;
  isPremiumTier?: boolean;
  toggleOSSettingsModal?: () => void;
  toggleBootstrapPackageModal?: () => void;
  hostSettings?: IHostMdmProfile[];
  showRefetchSpinner: boolean;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
  renderActionDropdown: () => JSX.Element | null;
  deviceUser?: boolean;
  osVersionRequirement?: IAppleDeviceUpdates;
  osSettings?: IOSSettings;
  hostMdmDeviceStatus?: HostMdmDeviceStatusUIState;
}

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

  if (platform === "rhel" || platform === "ubuntu") {
    return DISK_ENCRYPTION_MESSAGES.linux[
      diskEncryptionEnabled ? "enabled" : "unknown"
    ];
  }

  // mac or windows
  return DISK_ENCRYPTION_MESSAGES[platform][
    diskEncryptionEnabled ? "enabled" : "disabled"
  ];
};

const HostSummary = ({
  summaryData,
  bootstrapPackageData,
  isPremiumTier,
  toggleOSSettingsModal,
  toggleBootstrapPackageModal,
  hostSettings,
  showRefetchSpinner,
  onRefetchHost,
  renderActionDropdown,
  deviceUser,
  osVersionRequirement,
  osSettings,
  hostMdmDeviceStatus,
}: IHostSummaryProps): JSX.Element => {
  const {
    status,
    platform,
    os_version,
    disk_encryption_enabled: diskEncryptionEnabled,
  } = summaryData;

  const isChromeHost = platform === "chrome";
  const isIosOrIpadosHost = platform === "ios" || platform === "ipados";

  const renderRefetch = () => {
    const isOnline = summaryData.status === "online";
    let isDisabled = false;
    let tooltip;

    // we don't have a concept of "online" for iPads and iPhones, so always enable refetch
    if (!isIosOrIpadosHost) {
      // deviceStatus can be `undefined` in the case of the MyDevice Page not sending
      // this prop. When this is the case or when it is `unlocked`, we only take
      // into account the host being online or offline for correctly render the
      // refresh button. If we have a value for deviceStatus, we then need to also
      // take it account for rendering the button.
      if (
        hostMdmDeviceStatus === undefined ||
        hostMdmDeviceStatus === "unlocked"
      ) {
        isDisabled = !isOnline;
        tooltip = !isOnline ? REFETCH_TOOLTIP_MESSAGES.offline : null;
      } else {
        isDisabled = true;
        tooltip = !isOnline
          ? REFETCH_TOOLTIP_MESSAGES.offline
          : REFETCH_TOOLTIP_MESSAGES[hostMdmDeviceStatus];
      }
    }

    return (
      <RefetchButton
        isDisabled={isDisabled}
        isFetching={showRefetchSpinner}
        tooltip={tooltip}
        onRefetchHost={onRefetchHost}
      />
    );
  };

  const renderIssues = () => (
    <DataSet
      title="Issues"
      value={
        <IssuesIndicator
          totalIssuesCount={summaryData.issues.total_issues_count}
          criticalVulnerabilitiesCount={
            summaryData.issues.critical_vulnerabilities_count
          }
          failingPoliciesCount={summaryData.issues.failing_policies_count}
          tooltipPosition="bottom"
        />
      }
    />
  );

  const renderHostTeam = () => (
    <DataSet
      title="Team"
      value={
        summaryData.team_name !== "---" ? (
          `${summaryData.team_name}`
        ) : (
          <span className="no-team">No team</span>
        )
      }
    />
  );

  const renderDiskSpaceSummary = () => {
    return (
      <DataSet
        title="Disk space"
        value={
          <DiskSpaceIndicator
            baseClass="info-flex"
            gigsDiskSpaceAvailable={summaryData.gigs_disk_space_available}
            percentDiskSpaceAvailable={summaryData.percent_disk_space_available}
            id={`disk-space-tooltip-${summaryData.id}`}
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
        statusText = "Onx";
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

  const renderOperatingSystemSummary = () => {
    // No tooltip if minimum version is not set, including all Windows, Linux, ChromeOS operating systems
    if (!osVersionRequirement?.minimum_version) {
      return (
        <DataSet title="Operating system" value={summaryData.os_version} />
      );
    }

    const osVersionWithoutPrefix = removeOSPrefix(summaryData.os_version);
    const osVersionRequirementMet =
      compareVersions(
        osVersionWithoutPrefix,
        osVersionRequirement.minimum_version
      ) >= 0;

    return (
      <DataSet
        title="Operating system"
        value={
          <>
            {!osVersionRequirementMet && (
              <Icon name="error-outline" color="ui-fleet-black-75" />
            )}
            <TooltipWrapper
              tipContent={
                osVersionRequirementMet ? (
                  "Meets minimum version requirement."
                ) : (
                  <>
                    Does not meet minimum version requirement.
                    <br />
                    Deadline to update: {osVersionRequirement.deadline}
                  </>
                )
              }
            >
              {summaryData.os_version}
            </TooltipWrapper>
          </>
        }
      />
    );
  };

  const renderAgentSummary = () => {
    if (isChromeHost) {
      return <DataSet title="Agent" value={summaryData.osquery_version} />;
    }

    if (isIosOrIpadosHost) {
      return null;
    }

    if (summaryData.orbit_version !== DEFAULT_EMPTY_CELL_VALUE) {
      return (
        <DataSet
          title="Agent"
          value={
            <TooltipWrapper
              tipContent={
                <>
                  osquery: {summaryData.osquery_version}
                  <br />
                  Orbit: {summaryData.orbit_version}
                  {summaryData.fleet_desktop_version !==
                    DEFAULT_EMPTY_CELL_VALUE && (
                    <>
                      <br />
                      Fleet Desktop: {summaryData.fleet_desktop_version}
                    </>
                  )}
                </>
              }
            >
              {summaryData.orbit_version}
            </TooltipWrapper>
          }
        />
      );
    }
    return <DataSet title="Osquery" value={summaryData.osquery_version} />;
  };

  const renderMaintenanceWindow = ({
    starts_at,
    timezone,
  }: IHostMaintenanceWindow) => {
    const formattedStartsAt = formatInTimeZone(
      starts_at,
      // since startsAt is already localized and contains offset information, this 2nd parameter is
      // logically redundant. It's included here to allow use of date-fns-tz.formatInTimeZone instead of date-fns.format, which
      // allows us to format a UTC datetime without converting to the user-agent local time.
      timezone || "UTC",
      DATE_FNS_FORMAT_STRINGS.dateAtTime
    );

    const tip =
      timezone && timezone !== "UTC" ? (
        <>
          End user&apos;s time zone:
          <br />
          (GMT{starts_at.slice(-6)}) {timezone.replace("_", " ")}
        </>
      ) : (
        <>
          End user&apos;s timezone unavailable.
          <br />
          Displaying in UTC.
        </>
      );

    return (
      <DataSet
        title="Scheduled maintenance"
        value={
          <TooltipWrapper tipContent={tip}>{formattedStartsAt}</TooltipWrapper>
        }
      />
    );
  };

  const renderSummary = () => {
    // for windows and linux hosts we have to manually add a profile for disk encryption
    // as this is not currently included in the `profiles` value from the API
    // response for windows and linux hosts.
    if (
      platform === "windows" &&
      osSettings?.disk_encryption?.status &&
      isWindowsDiskEncryptionStatus(osSettings.disk_encryption.status)
    ) {
      const winDiskEncryptionSetting: IHostMdmProfile = generateWinDiskEncryptionSetting(
        osSettings.disk_encryption.status,
        osSettings.disk_encryption.detail
      );
      hostSettings = hostSettings
        ? [...hostSettings, winDiskEncryptionSetting]
        : [winDiskEncryptionSetting];
    }

    if (
      isDiskEncryptionSupportedLinuxPlatform(platform, os_version) &&
      osSettings?.disk_encryption?.status &&
      isLinuxDiskEncryptionStatus(osSettings.disk_encryption.status)
    ) {
      const linuxDiskEncryptionSetting: IHostMdmProfile = generateLinuxDiskEncryptionSetting(
        osSettings.disk_encryption.status,
        osSettings.disk_encryption.detail
      );
      hostSettings = hostSettings
        ? [...hostSettings, linuxDiskEncryptionSetting]
        : [linuxDiskEncryptionSetting];
    }

    return (
      <Card
        borderRadiusSize="xxlarge"
        includeShadow
        largePadding
        className={`${baseClass}-card`}
      >
        {!isIosOrIpadosHost && (
          <DataSet
            title="Status"
            value={
              <StatusIndicator
                value={status || ""} // temporary work around of integration test bug
                tooltip={{
                  tooltipText: getHostStatusTooltipText(status),
                  position: "bottom",
                }}
              />
            }
          />
        )}
        {summaryData.issues?.total_issues_count > 0 &&
          !isIosOrIpadosHost &&
          renderIssues()}
        {isPremiumTier && renderHostTeam()}
        {/* Rendering of OS Settings data */}
        {isOsSettingsDisplayPlatform(platform, os_version) &&
          isPremiumTier &&
          hostSettings &&
          hostSettings.length > 0 && (
            <DataSet
              title="OS settings"
              value={
                <OSSettingsIndicator
                  profiles={hostSettings}
                  onClick={toggleOSSettingsModal}
                />
              }
            />
          )}
        {bootstrapPackageData?.status && !isIosOrIpadosHost && (
          <DataSet
            title="Bootstrap package"
            value={
              <BootstrapPackageIndicator
                status={bootstrapPackageData.status}
                onClick={toggleBootstrapPackageModal}
              />
            }
          />
        )}
        {!isChromeHost && renderDiskSpaceSummary()}
        {renderDiskEncryptionSummary()}
        {!isIosOrIpadosHost && (
          <DataSet
            title="Memory"
            value={wrapFleetHelper(humanHostMemory, summaryData.memory)}
          />
        )}
        {!isIosOrIpadosHost && (
          <DataSet title="Processor type" value={summaryData.cpu_type} />
        )}
        {renderOperatingSystemSummary()}
        {!isIosOrIpadosHost && renderAgentSummary()}
        {isPremiumTier &&
          // TODO - refactor normalizeEmptyValues pattern
          !!summaryData.maintenance_window &&
          summaryData.maintenance_window !== "---" &&
          renderMaintenanceWindow(summaryData.maintenance_window)}
      </Card>
    );
  };

  const lastFetched = summaryData.detail_updated_at ? (
    <HumanTimeDiffWithFleetLaunchCutoff
      timeString={summaryData.detail_updated_at}
    />
  ) : (
    ": unavailable"
  );

  const renderDeviceStatusTag = () => {
    if (!hostMdmDeviceStatus || hostMdmDeviceStatus === "unlocked") return null;

    const tag = DEVICE_STATUS_TAGS[hostMdmDeviceStatus];

    const classNames = classnames(
      `${baseClass}__device-status-tag`,
      tag.tagType
    );

    return (
      <>
        <span className={classNames} data-tip data-for="tag-tooltip">
          {tag.title}
        </span>
        <ReactTooltip
          place="top"
          effect="solid"
          id="tag-tooltip"
          backgroundColor={COLORS["tooltip-bg"]}
        >
          <span className={`${baseClass}__tooltip-text`}>
            {tag.generateTooltip(platform)}
          </span>
        </ReactTooltip>
      </>
    );
  };

  return (
    <div className={baseClass}>
      <div className="header title">
        <div className="title__inner">
          <div className="display-name-container">
            <h1 className="display-name">
              {deviceUser
                ? "My device"
                : summaryData.display_name || DEFAULT_EMPTY_CELL_VALUE}
            </h1>

            {renderDeviceStatusTag()}

            <div className={`${baseClass}__last-fetched`}>
              {"Last fetched"} {lastFetched}
              &nbsp;
            </div>
            {renderRefetch()}
          </div>
        </div>
        {renderActionDropdown()}
      </div>
      {renderSummary()}
    </div>
  );
};

export default HostSummary;
