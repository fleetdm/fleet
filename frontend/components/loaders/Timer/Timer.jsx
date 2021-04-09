import React, { Component } from "react";
import PropTypes from "prop-types";

import { convertSeconds } from "./helpers";

const baseClass = "kolide-timer";

class Timer extends Component {
  static propTypes = {
    totalMilliseconds: PropTypes.number,
  };

  render() {
    const { totalMilliseconds } = this.props;

    return (
      <span className={baseClass}>{convertSeconds(totalMilliseconds)}</span>
    );
  }
}

export default Timer;
