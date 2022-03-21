import React from "react";

import ReactTooltip from "react-tooltip";

import Button from "components/buttons/Button";
import { humanHostMemory } from "fleet/helpers";
import IssueIcon from "../../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

const baseClass = "host-summary";

interface IHostSummaryProps {
  statusClassName: string;
  titleData: any;
  isPremiumTier?: boolean;
  wrapFleetHelper: (helperFn: (value: any) => string, value: string) => string;
  isOnlyObserver?: boolean;
  toggleOSPolicyModal?: () => void;
  deviceUser?: boolean;
}

const HostSummary = ({
  statusClassName,
  titleData,
  isPremiumTier,
  wrapFleetHelper,
  isOnlyObserver,
  toggleOSPolicyModal,
  deviceUser,
}: IHostSummaryProps): JSX.Element => {
  const renderIssues = () => (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">Issues</span>
      <span className="info-flex__data">
        <span
          className="host-issue tooltip__tooltip-icon"
          data-tip
          data-for="host-issue-count"
          data-tip-disable={false}
        >
          <img alt="host issue" src={IssueIcon} />
        </span>
        <ReactTooltip
          place="bottom"
          type="dark"
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

  const renderDiskSpace = () => {
    if (
      titleData &&
      (titleData.gigs_disk_space_available > 0 ||
        titleData.percent_disk_space_available > 0)
    ) {
      return (
        <span className="info-flex__data">
          <div className="info-flex__disk-space">
            <div
              className={
                titleData.percent_disk_space_available > 20
                  ? "info-flex__disk-space-used"
                  : "info-flex__disk-space-warning"
              }
              style={{
                width: `${100 - titleData.percent_disk_space_available}%`,
              }}
            />
          </div>
          {titleData.gigs_disk_space_available} GB available
        </span>
      );
    }
    return <span className="info-flex__data">No data available</span>;
  };

  if (deviceUser) {
    return (
      <div className="section title">
        <div className="title__inner">
          <div className="info-flex">
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">Status</span>
              <span className={`${statusClassName} info-flex__data`}>
                {titleData.status}
              </span>
            </div>
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">Disk Space</span>
              {renderDiskSpace()}
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
              <span className="info-flex__data">{titleData.os_version}</span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="section title">
      <div className="title__inner">
        <div className="info-flex">
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">Status</span>
            <span className={`${statusClassName} info-flex__data`}>
              {titleData.status}
            </span>
          </div>
          {titleData.issues?.total_issues_count > 0 && renderIssues()}
          {isPremiumTier && renderHostTeam()}
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">Disk Space</span>
            {renderDiskSpace()}
          </div>
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">RAM</span>
            <span className="info-flex__data">
              {wrapFleetHelper(humanHostMemory, titleData.memory)}
            </span>
          </div>
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">CPU</span>
            <span className="info-flex__data">{titleData.cpu_type}</span>
          </div>
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">OS</span>
            <span className="info-flex__data">
              {isOnlyObserver ? (
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
          <div className="info-flex__item info-flex__item--title">
            <span className="info-flex__header">Osquery</span>
            <span className="info-flex__data">{titleData.osquery_version}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default HostSummary;
