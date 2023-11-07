import React from "react";
import ReactTooltip from "react-tooltip";

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
import {
  getHostDiskEncryptionTooltipMessage,
  humanHostMemory,
  wrapFleetHelper,
} from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import StatusIndicator from "components/StatusIndicator";

import OSSettingsIndicator from "./OSSettingsIndicator";
import HostSummaryIndicator from "./HostSummaryIndicator";
import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";
import { generateWinDiskEncryptionProfile } from "../../helpers";

const baseClass = "host-summary";

interface IBootstrapPackageData {
  status?: BootstrapPackageStatus | "";
  details?: string;
}

interface IHostSummaryProps {
  titleData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  bootstrapPackageData?: IBootstrapPackageData;
  diskEncryptionEnabled?: boolean;
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
}

const HostSummary = ({
  titleData,
  bootstrapPackageData,
  diskEncryptionEnabled,
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
}: IHostSummaryProps): JSX.Element => {
  const { status, platform } = titleData;

  const renderRefetch = () => {
    const isOnline = titleData.status === "online";

    return (
      <>
        <div
          className="refetch"
          data-tip
          data-for="refetch-tooltip"
          data-tip-disable={isOnline || showRefetchSpinner}
        >
          <Button
            className={`
            button
            ${!isOnline ? "refetch-offline tooltip" : ""}
              ${showRefetchSpinner ? "refetch-spinner" : "refetch-btn"}
            `}
            disabled={!isOnline}
            onClick={onRefetchHost}
            variant="text-icon"
          >
            <Icon name="refresh" color="core-fleet-blue" size="small" />
            {showRefetchSpinner
              ? "Fetching fresh vitals...this may take a moment"
              : "Refetch"}
          </Button>
        </div>
        <ReactTooltip
          place="top"
          effect="solid"
          id="refetch-tooltip"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t fetch data from <br /> an offline host.
          </span>
        </ReactTooltip>
      </>
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
          backgroundColor="#3e4771"
          id="host-issue-count"
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Failing policies ({titleData.issues.failing_policies_count})
          </span>
        </ReactTooltip>
        <span className={"info-flex__data__text"}>
          {titleData.issues.total_issues_count}
        </span>
      </span>
    </div>
  );

  const renderHostTeam = () => (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">Team</span>
      <span className={`info-flex__data`}>
        {titleData.team_name ? (
          `${titleData.team_name}`
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
    if (platform === "chrome") {
      statusText = "Always on";
    } else {
      statusText = diskEncryptionEnabled ? "On" : "Off";
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

        {(titleData.issues?.total_issues_count > 0 || isSandboxMode) &&
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
              gigsDiskSpaceAvailable={titleData.gigs_disk_space_available}
              percentDiskSpaceAvailable={titleData.percent_disk_space_available}
              id={`disk-space-tooltip-${titleData.id}`}
              platform={platform}
              tooltipPosition="bottom"
            />
          </div>
        )}

        {renderDiskEncryptionSummary()}

        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Memory</span>
          <span className="info-flex__data">
            {wrapFleetHelper(humanHostMemory, titleData.memory)}
          </span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Processor type</span>
          <span className="info-flex__data">{titleData.cpu_type}</span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Operating system</span>
          <span className="info-flex__data">{titleData.os_version}</span>
        </div>
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Osquery</span>
          <span className="info-flex__data">{titleData.osquery_version}</span>
        </div>
      </div>
    );
  };

  const lastFetched = titleData.detail_updated_at ? (
    <HumanTimeDiffWithFleetLaunchCutoff
      timeString={titleData.detail_updated_at}
    />
  ) : (
    ": unavailable"
  );

  return (
    <div className={baseClass}>
      <div className="header title">
        <div className="title__inner">
          <div className="display-name-container">
            <h1 className="display-name">
              {deviceUser
                ? "My device"
                : titleData.display_name || DEFAULT_EMPTY_CELL_VALUE}
            </h1>

            <p className="last-fetched">
              {"Last fetched"} {lastFetched}
              &nbsp;
            </p>
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
