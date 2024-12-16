import React from "react";
import { noop } from "lodash";
import AceEditor from "react-ace";
import classnames from "classnames";

import { humanHostMemory } from "utilities/helpers";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
// @ts-ignore
import PlatformIcon from "components/icons/PlatformIcon";
import { ISelectHost, ISelectLabel, ISelectTeam } from "interfaces/target";

import { isTargetHost, isTargetTeam, isTargetLabel } from "../helpers";

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
  const onlineHosts = (
    labelBaseClass: string,
    count: number,
    online: number
  ) => {
    const offline = count - online;
    const percentCount = ((count - offline) / count) * 100;
    const percentOnline = parseFloat(percentCount.toFixed(2));

    if (online > 0) {
      return (
        <span className={`${labelBaseClass}__hosts-online`}>
          {" "}
          ({percentOnline}% ONLINE)
        </span>
      );
    }

    return false;
  };

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

        <p className={`${hostBaseClass}__display-text`}>
          <FleetIcon name="single-host" className={`${hostBaseClass}__icon`} />
          <span>{displayText}</span>
        </p>
        <p className={statusClassName}>
          {isOnline && (
            <FleetIcon
              name="success-check"
              className={`${hostBaseClass}__icon ${hostBaseClass}__icon--online`}
            />
          )}
          {isOffline && (
            <FleetIcon
              name="offline"
              className={`${hostBaseClass}__icon ${hostBaseClass}__icon--offline`}
            />
          )}
          <span>{status}</span>
        </p>
        <table className={`${baseClass}__table`}>
          <tbody>
            <tr>
              <th>Private IP address</th>
              <td>{hostIpAddress}</td>
            </tr>
            <tr>
              <th>MAC address</th>
              <td>
                <span className={`${hostBaseClass}__mac-address`}>
                  {hostMac}
                </span>
              </td>
            </tr>
            <tr>
              <th>Platform</th>
              <td>
                <PlatformIcon name={platform} title={platform} />
                <span className={`${hostBaseClass}__platform-text`}>
                  {" "}
                  {platform}
                </span>
              </td>
            </tr>
            <tr>
              <th>Operating system</th>
              <td>{osVersion}</td>
            </tr>
            <tr>
              <th>Osquery version</th>
              <td>{osqueryVersion}</td>
            </tr>
            <tr>
              <th>Memory</th>
              <td>{humanHostMemory(memory)}</td>
            </tr>
          </tbody>
        </table>
      </div>
    );
  };

  const renderLabel = (labelTarget: ISelectLabel) => {
    const {
      count,
      description,
      display_text: displayText,
      label_type: labelType,
      // online,
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
          <span className={`${labelBaseClass}__hosts-count`}>
            <strong>{count}</strong>HOSTS
          </span>
          {/* {onlineHosts(labelBaseClass, count, online)} */}
        </p>

        <p className={`${labelBaseClass}__description`}>
          {description || "No Description"}
        </p>
        <div className={`${labelBaseClass}__editor`}>
          <AceEditor
            editorProps={{ $blockScrolling: Infinity }}
            mode="fleet"
            minLines={1}
            maxLines={20}
            name="label-query"
            readOnly
            setOptions={{ wrap: true }}
            showGutter={false}
            showPrintMargin={false}
            theme="fleet"
            value={query}
            width="100%"
            fontSize={14}
          />
        </div>
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
          <span className={`${labelBaseClass}__hosts-count`}>
            <strong>{count}</strong>HOSTS
          </span>
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
