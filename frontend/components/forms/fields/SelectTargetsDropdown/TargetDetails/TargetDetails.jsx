import React, { Component } from "react";
import PropTypes from "prop-types";
import { noop } from "lodash";
import AceEditor from "react-ace";
import classnames from "classnames";

import { humanHostMemory } from "fleet/helpers";
import FleetIcon from "components/icons/FleetIcon";
import PlatformIcon from "components/icons/PlatformIcon";
import targetInterface from "interfaces/target";

const baseClass = "target-details";

class TargetDetails extends Component {
  static propTypes = {
    target: targetInterface,
    className: PropTypes.string,
    handleBackToResults: PropTypes.func,
  };

  static defaultProps = {
    handleBackToResults: noop,
  };

  onlineHosts = (labelBaseClass, count, online) => {
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

  renderHost = () => {
    const { className, handleBackToResults, target } = this.props;
    const {
      display_text: displayText,
      primary_mac: hostMac,
      primary_ip: hostIpAddress,
      memory,
      osquery_version: osqueryVersion,
      os_version: osVersion,
      platform,
      status,
    } = target;
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
              <th>IP Address</th>
              <td>{hostIpAddress}</td>
            </tr>
            <tr>
              <th>MAC Address</th>
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
              <th>Operating System</th>
              <td>{osVersion}</td>
            </tr>
            <tr>
              <th>Osquery Version</th>
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

  renderLabel = () => {
    const { onlineHosts } = this;
    const { handleBackToResults, className, target } = this.props;
    const {
      count,
      description,
      display_text: displayText,
      label_type: labelType,
      online,
      query,
    } = target;
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
          {onlineHosts(labelBaseClass, count, online)}
        </p>

        <p className={`${labelBaseClass}__description`}>
          {description || "No Description"}
        </p>

        {labelType !== 1 && (
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
        )}
      </div>
    );
  };

  render() {
    const { target } = this.props;

    if (!target) {
      return false;
    }

    const { target_type: targetType } = target;
    const { renderHost, renderLabel } = this;

    if (targetType === "labels") {
      return renderLabel();
    }

    return renderHost();
  }
}

export default TargetDetails;
