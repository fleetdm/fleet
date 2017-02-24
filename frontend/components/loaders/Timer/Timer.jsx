import React, { Component, PropTypes } from 'react';

import { convertSeconds } from './helpers';

const baseClass = 'kolide-timer';

class Timer extends Component {
  static propTypes = {
    totalMilliseconds: PropTypes.number,
  }

  render () {
    const { totalMilliseconds } = this.props;

    return (
      <span className={baseClass}>{convertSeconds(totalMilliseconds)}</span>
    );
  }
}

export default Timer;
