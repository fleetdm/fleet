import React from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";

import {
  IHostMdmProfile,
  BootstrapPackageStatus,
  isWindowsDiskEncryptionStatus,
} from "interfaces/mdm";
import { IOSSettings } from "interfaces/host";
import getHostStatusTooltipText from "pages/hosts/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import DiskSpaceGraph from "components/DiskSpaceGraph";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import { humanHostMemory, wrapFleetHelper } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import StatusIndicator from "components/StatusIndicator";
import { COLORS } from "styles/var/colors";

import OSSettingsIndicator from "./OSSettingsIndicator";
import HostSummaryIndicator from "./HostSummaryIndicator";
import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";

import {
  HostMdmDeviceStatusUIState,
  generateWinDiskEncryptionProfile,
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
  const conditionalProps: any = {};
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
  isSandboxMode?: boolean;
  toggleOSSettingsModal?: () => void;
  toggleBootstrapPackageModal?: () => void;
  hostMdmProfiles?: IHostMdmProfile[];
  mdmName?: string;
  showRefetchSpinner: boolean;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
  renderActionButtons: () => JSX.Element | null;
  deviceUser?: boolean;
  osSettings?: IOSSettings;
  hostMdmDeviceStatus?: HostMdmDeviceStatusUIState;
}

const MAC_WINDOWS_DISK_ENCRYPTION_MESSAGES = {
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
};

const getHostDiskEncryptionTooltipMessage = (
  platform: "darwin" | "windows" | "chrome", // TODO: improve this type
  diskEncryptionEnabled = false
) => {
  if (platform === "chrome") {
    return "Fleet does not check for disk encryption on Chromebooks, as they are encrypted by default.";
  }

  if (!["windows", "darwin"].includes(platform)) {
    return "Disk encryption is enabled.";
  }
  return MAC_WINDOWS_DISK_ENCRYPTION_MESSAGES[platform][
    diskEncryptionEnabled ? "enabled" : "disabled"
  ];
};

const HostSummary = ({
  summaryData,
  bootstrapPackageData,
  isPremiumTier,
  isSandboxMode = false,
  toggleOSSettingsModal,
  toggleBootstrapPackageModal,
  hostMdmProfiles,
  mdmName,
  showRefetchSpinner,
  onRefetchHost,
  renderActionButtons,
  deviceUser,
  osSettings,
  hostMdmDeviceStatus,
}: IHostSummaryProps): JSX.Element => {
  const {
    status,
    platform,
    disk_encryption_enabled: diskEncryptionEnabled,
  } = summaryData;

  const renderRefetch = () => {
    const isOnline = summaryData.status === "online";
    let isDisabled = false;
    let tooltip: React.ReactNode = <></>;

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
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">
        Issues{isSandboxMode && <PremiumFeatureIconWithTooltip />}
      </span>
      <span className="info-flex__data">
        <span
          className="host-issue tooltip tooltip__tooltip-icon"
          data-tip
          data-for="host-issue-count"
          data-tip-disable={false}
        >
          <Icon name="error-outline" color="ui-fleet-black-50" />
        </span>
        <ReactTooltip
          place="bottom"
          effect="solid"
          backgroundColor={COLORS["tooltip-bg"]}
          id="host-issue-count"
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Failing policies ({summaryData.issues.failing_policies_count})
          </span>
        </ReactTooltip>
        <span className="info-flex__data__text">
          {summaryData.issues.total_issues_count}
        </span>
      </span>
    </div>
  );

  const renderHostTeam = () => (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">Team</span>
      <span className={`info-flex__data`}>
        {summaryData.team_name ? (
          `${summaryData.team_name}`
        ) : (
          <span className="info-flex__no-team">No team</span>
        )}
      </span>
    </div>
  );

  const renderDiskEncryptionSummary = () => {
    // TODO: improve this typing, platforms!
    if (!["darwin", "windows", "chrome"].includes(platform)) {
      return <></>;
    }
    const tooltipMessage = getHostDiskEncryptionTooltipMessage(
      platform,
      diskEncryptionEnabled
    );

    let statusText;
    switch (true) {
      case platform === "chrome":
        statusText = "Always on";
        break;
      case diskEncryptionEnabled === true:
        statusText = "On";
        break;
      case diskEncryptionEnabled === false:
        statusText = "Off";
        break;
      default:
        // something unexpected happened on the way to this component, display whatever we got or
        // "Unknown" to draw attention to the issue.
        statusText = diskEncryptionEnabled || "Unknown";
    }

    return (
      <div className="info-flex__item info-flex__item--title">
        <span className="info-flex__header">Disk encryption</span>
        <TooltipWrapper tipContent={tooltipMessage}>
          {statusText}
        </TooltipWrapper>
      </div>
    );
  };

  const renderSummary = () => {
    // for windows hosts we have to manually add a profile for disk encryption
    // as this is not currently included in the `profiles` value from the API
    // response for windows hosts.
    if (
      platform === "windows" &&
      osSettings?.disk_encryption?.status &&
      isWindowsDiskEncryptionStatus(osSettings.disk_encryption.status)
    ) {
      const winDiskEncryptionProfile: IHostMdmProfile = generateWinDiskEncryptionProfile(
        osSettings.disk_encryption.status,
        osSettings.disk_encryption.detail
      );
      hostMdmProfiles = hostMdmProfiles
        ? [...hostMdmProfiles, winDiskEncryptionProfile]
        : [winDiskEncryptionProfile];
    }

    return (
      <div className="info-flex">
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Status</span>
          <StatusIndicator
            value={status || ""} // temporary work around of integration test bug
            tooltip={{
              tooltipText: getHostStatusTooltipText(status),
              position: "bottom",
            }}
          />
        </div>

        {(summaryData.issues?.total_issues_count > 0 || isSandboxMode) &&
          isPremiumTier &&
          renderIssues()}

        {isPremiumTier && renderHostTeam()}

        {/* Rendering of OS Settings data */}
        {(platform === "darwin" || platform === "windows") &&
          isPremiumTier &&
          // TODO: API INTEGRATION: change this when we figure out why the API is
          // returning "Fleet" or "FleetDM" for the MDM name.
          mdmName?.includes("Fleet") && // show if 1 - host is enrolled in Fleet MDM, and
          hostMdmProfiles &&
          hostMdmProfiles.length > 0 && ( // 2 - host has at least one setting (profile) enforced
            <HostSummaryIndicator title="OS settings">
              <OSSettingsIndicator
                profiles={hostMdmProfiles}
                onClick={toggleOSSettingsModal}
              />
            </HostSummaryIndicator>
          )}

        {bootstrapPackageData?.status && (
          <HostSummaryIndicator title="Bootstrap package">
            <BootstrapPackageIndicator
              status={bootstrapPackageData.status}
              onClick={toggleBootstrapPackageModal}
            />
          </HostSummaryIndicator>
        )}

        {platform !== "chrome" && (
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">Disk space</span>
            <DiskSpaceGraph
              baseClass="info-flex"
              gigsDiskSpaceAvailable={summaryData.gigs_disk_space_available}
              percentDiskSpaceAvailable={
                summaryData.percent_disk_space_available
              }
              id={`disk-space-tooltip-${summaryData.id}`}
              platform={platform}
              tooltipPosition="bottom"
            />
          </div>
        )}

        {renderDiskEncryptionSummary()}

        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Memory</span>
          <span className="info-flex__data">
            {wrapFleetHelper(humanHostMemory, summaryData.memory)}
          </span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Processor type</span>
          <span className="info-flex__data">{summaryData.cpu_type}</span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Operating system</span>
          <span className="info-flex__data">{summaryData.os_version}</span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Osquery</span>
          <span className="info-flex__data">{summaryData.osquery_version}</span>
        </div>
      </div>
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
        {renderActionButtons()}
      </div>
      <div className="section title">
        <div className="title__inner">{renderSummary()}</div>
      </div>
    </div>
  );
};

export default HostSummary;
