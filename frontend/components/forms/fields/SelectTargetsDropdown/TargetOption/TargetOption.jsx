import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import targetInterface from "interfaces/target";
import TargetIcon from "./TargetIcon";

const baseClass = "target-option";

class TargetOption extends Component {
  static propTypes = {
    onMoreInfoClick: PropTypes.func,
    onSelect: PropTypes.func,
    target: targetInterface.isRequired,
  };

  handleSelect = (evt) => {
    const { onSelect, target } = this.props;

    return onSelect(target, evt);
  };

  renderTargetDetail = () => {
    const { target } = this.props;
    const {
      count,
      primary_ip: hostIpAddress,
      target_type: targetType,
    } = target;

    if (targetType === "hosts") {
      if (!hostIpAddress) {
        return false;
      }

      return (
        <span>
          <span className={`${baseClass}__ip`}>{hostIpAddress}</span>
        </span>
      );
    }

    return <span className={`${baseClass}__count`}>{count} hosts</span>;
  };

  render() {
    const { onMoreInfoClick, target } = this.props;
    const { display_text: displayText, target_type: targetType } = target;
    const { handleSelect, renderTargetDetail } = this;
    const wrapperClassName = classnames(`${baseClass}__wrapper`, {
      "is-team": targetType === "teams",
      "is-label": targetType === "labels",
      "is-host": targetType === "hosts",
    });

    return (
      <div className={wrapperClassName}>
        <button
          className={`button button--unstyled ${baseClass}__target-content`}
          onClick={onMoreInfoClick(target)}
        >
          <div>
            <TargetIcon target={target} />
            <span className={`${baseClass}__label-label`}>
              {displayText !== "All Hosts" ? displayText : "All hosts"}
            </span>
          </div>
          {renderTargetDetail()}
        </button>
        <button
          className={`button button--unstyled ${baseClass}__add-btn`}
          onClick={handleSelect}
        />
      </div>
    );
  }
}

export default TargetOption;
