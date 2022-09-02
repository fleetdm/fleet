import React from "react";

import ReactTooltip from "react-tooltip";

import Button from "components/buttons/Button";
import DiskSpaceGraph from "components/DiskSpaceGraph";
import {
  humanHostMemory,
  humanHostDetailUpdated,
  wrapFleetHelper,
} from "utilities/helpers";
import IssueIcon from "../../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

const baseClass = "host-summary";

interface IHostSummaryProps {
  statusClassName: string;
  titleData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  isPremiumTier?: boolean;
  isOnlyObserver?: boolean;
  toggleOSPolicyModal?: () => void;
  showRefetchSpinner: boolean;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
  renderActionButtons: () => JSX.Element;
  deviceUser?: boolean;
}

const HostSummary = ({
  statusClassName,
  titleData,
  isPremiumTier,
  isOnlyObserver,
  toggleOSPolicyModal,
  showRefetchSpinner,
  onRefetchHost,
  renderActionButtons,
  deviceUser,
}: IHostSummaryProps): JSX.Element => {
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
            {showRefetchSpinner
              ? "Fetching fresh vitals...this may take a moment"
              : "Refetch"}
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
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
      <span className="info-flex__header">Issues</span>
      <span className="info-flex__data">
        <span
          className="host-issue tooltip tooltip__tooltip-icon"
          data-tip
          data-for="host-issue-count"
          data-tip-disable={false}
        >
          <img alt="host issue" src={IssueIcon} />
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
        <span className={`total-issues-count`}>
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

  const renderSummary = () => {
    return (
      <div className="info-flex">
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Status</span>
          <span className={`${statusClassName} info-flex__data`}>
            {titleData.status}
          </span>
        </div>
        {titleData.issues?.total_issues_count > 0 &&
          deviceUser &&
          isPremiumTier &&
          renderIssues()}
        {titleData.issues?.total_issues_count > 0 &&
          !deviceUser &&
          renderIssues()}
        {!deviceUser && isPremiumTier && renderHostTeam()}
        <div className="info-flex__item info-flex__item--title">
          <span className="info-flex__header">Disk space</span>
          <DiskSpaceGraph
            baseClass="info-flex"
            gigsDiskSpaceAvailable={titleData.gigs_disk_space_available}
            percentDiskSpaceAvailable={titleData.percent_disk_space_available}
            id={"disk-space-tooltip"}
          />
        </div>
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
          <span className="info-flex__data">
            {isOnlyObserver || deviceUser ? (
              `${titleData.os_version}`
            ) : (
              <Button
                onClick={() => toggleOSPolicyModal && toggleOSPolicyModal()}
                variant="text-link"
                className={`${baseClass}__os-policy-button`}
              >
                {titleData.os_version}
              </Button>
            )}
          </span>
        </div>
        {!deviceUser && (
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">Osquery</span>
            <span className="info-flex__data">{titleData.osquery_version}</span>
          </div>
        )}
      </div>
    );
  };

  return (
    <>
      <div className="header title">
        <div className="title__inner">
          <div className="display-name-container">
            <h1 className="display-name">
              {deviceUser ? "My device" : titleData.display_name || "---"}
            </h1>
            <p className="last-fetched">
              {`Last fetched ${humanHostDetailUpdated(
                titleData.detail_updated_at
              )}`}
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
    </>
  );
};

export default HostSummary;
