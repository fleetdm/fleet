import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { round } from "lodash";

const baseClass = "progress-bar";

class ProgressBar extends Component {
  static propTypes = {
    className: PropTypes.string,
    error: PropTypes.number.isRequired,
    max: PropTypes.number.isRequired,
    success: PropTypes.number.isRequired,
  };

  render() {
    const { className, error, max, success } = this.props;
    const successPercentComplete = `${round((success / (max || 1)) * 100, 0)}%`;
    const errorPercentComplete = `${round((error / (max || 1)) * 100, 0)}%`;
    const wrapperClassName = classnames(baseClass, className);

    return (
      <div className={wrapperClassName}>
        <div
          className={`${baseClass}__progress ${baseClass}__progress--success`}
          style={{ width: successPercentComplete }}
        />
        <div
          className={`${baseClass}__progress ${baseClass}__progress--error`}
          style={{ width: errorPercentComplete }}
        />
      </div>
    );
  }
}

export default ProgressBar;
