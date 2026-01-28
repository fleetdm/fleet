import React from "react";
import { noop } from "lodash";
import SQLEditor from "components/SQLEditor";
import classnames from "classnames";

import { humanHostMemory } from "utilities/helpers";
import FleetIcon from "components/icons/FleetIcon";
import OSIcon from "pages/SoftwarePage/components/icons/OSIcon";
import { ISelectHost, ISelectLabel, ISelectTeam } from "interfaces/target";
import DataSet from "components/DataSet";
import StatusIndicator from "components/StatusIndicator";

import { isTargetHost, isTargetTeam, isTargetLabel } from "../helpers";
import TableCount from "components/TableContainer/TableCount";

const baseClass = "target-details";

interface ITargetDetailsProps {
  target: ISelectHost | ISelectTeam | ISelectLabel; // Replace with Target
  className?: string;
  handleBackToResults?: () => void;
}

const TargetDetails = ({
  target,
  className = "",
  handleBackToResults = noop,
}: ITargetDetailsProps): JSX.Element => {
  const renderHost = (hostTarget: ISelectHost) => {
    const {
      display_text: displayText,
      primary_mac: hostMac,
      primary_ip: hostIpAddress,
      memory,
      osquery_version: osqueryVersion,
      os_version: osVersion,
      platform,
      status,
    } = hostTarget;
    const hostBaseClass = "host-target";
    const isOnline = status === "online";
    const isOffline = status === "offline";
    const statusClassName = classnames(
      `${hostBaseClass}__status`,
      { [`${hostBaseClass}__status--is-online`]: isOnline },
      { [`${hostBaseClass}__status--is-offline`]: isOffline }
    );

    return (
      <div className={`${hostBaseClass} ${className}`}>
        <button
          className={`button button--unstyled ${hostBaseClass}__back`}
          onClick={handleBackToResults}
        >
          <FleetIcon name="chevronleft" />
          Back
        </button>

        <div className={`${hostBaseClass}__host-info`}>
          <div className={`${hostBaseClass}__display-text`}>
            <FleetIcon
              name="single-host"
              className={`${hostBaseClass}__icon`}
            />
            <span>{displayText}</span>
          </div>
          <div className={statusClassName}>
            {isOnline && <StatusIndicator value="online" />}
            {isOffline && <StatusIndicator value="offline" />}
          </div>
        </div>
        <div className={`${baseClass}__details`}>
          <DataSet
            title="Private IP address"
            value={hostIpAddress}
            orientation="horizontal"
          />
          <DataSet
            title="MAC address"
            value={
              <span className={`${hostBaseClass}__mac-address`}>{hostMac}</span>
            }
            orientation="horizontal"
          />
          <DataSet
            title="Platform"
            value={
              <>
                <OSIcon name={platform} />
                <span className={`${hostBaseClass}__platform-text`}>
                  {" "}
                  {platform}
                </span>
              </>
            }
            orientation="horizontal"
          />
          <DataSet
            title="Operating system"
            value={osVersion}
            orientation="horizontal"
          />
          <DataSet
            title="Osquery version"
            value={osqueryVersion}
            orientation="horizontal"
          />
          <DataSet
            title="Memory"
            value={humanHostMemory(memory)}
            orientation="horizontal"
          />
        </div>
      </div>
    );
  };

  const renderLabel = (labelTarget: ISelectLabel) => {
    const {
      count,
      description,
      display_text: displayText,
      query,
    } = labelTarget;

    const labelBaseClass = "label-target";
    return (
      <div className={`${labelBaseClass} ${className}`}>
        <button
          className={`button button--unstyled ${labelBaseClass}__back`}
          onClick={handleBackToResults}
        >
          <FleetIcon name="chevronleft" /> Back
        </button>

        <p className={`${labelBaseClass}__display-text`}>
          <FleetIcon name="label" fw className={`${labelBaseClass}__icon`} />
          <span>{displayText}</span>
        </p>

        <p className={`${labelBaseClass}__hosts`}>
          <TableCount count={count} name="hosts" />
        </p>

        <p className={`${labelBaseClass}__description`}>
          {description || "No Description"}
        </p>
        {query && (
          <div className={`${labelBaseClass}__editor`}>
            <SQLEditor
              name="label-query"
              value={query}
              readOnly
              disabled
              maxLines={20}
              showGutter={false}
              wrapEnabled
              fontSize={14}
              style={{ width: "100%" }}
            />
          </div>
        )}
      </div>
    );
  };

  const renderTeam = (teamTarget: ISelectTeam) => {
    const { count, display_text: displayText } = teamTarget;
    const labelBaseClass = "label-target";

    return (
      <div className={`${labelBaseClass} ${className}`}>
        <p className={`${labelBaseClass}__display-text`}>
          <FleetIcon
            name="all-hosts"
            fw
            className={`${labelBaseClass}__icon`}
          />
          <span>{displayText}</span>
        </p>

        <p className={`${labelBaseClass}__hosts`}>
          <TableCount count={count} name="hosts" />
        </p>
      </div>
    );
  };

  if (!target) {
    return <></>;
  }

  if (isTargetHost(target)) {
    return renderHost(target);
  }

  if (isTargetLabel(target)) {
    return renderLabel(target);
  }

  if (isTargetTeam(target)) {
    return renderTeam(target);
  }
  return <></>;
};

export default TargetDetails;
