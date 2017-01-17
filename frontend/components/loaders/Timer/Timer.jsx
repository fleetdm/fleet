import React, { Component, PropTypes } from 'react';

import { convertSeconds } from './helpers';

const baseClass = 'kolide-timer';

class Timer extends Component {
  static propTypes = {
    running: PropTypes.bool,
  }

  constructor (props) {
    super(props);

    this.state = { totalMilliseconds: 0 };
  }

  componentWillReceiveProps ({ running }) {
    const { running: currentRunning } = this.props;

    if (running) {
      if (!currentRunning) {
        this.reset();
      }

      this.play();
    } else {
      this.pause();
    }
  }

  componentWillUnmount () {
    this.pause();
  }

  play = () => {
    const { interval, update } = this;

    if (!interval) {
      this.interval = setInterval(update, 1000);
    }
  }

  pause = () => {
    const { interval } = this;

    if (interval) {
      clearInterval(interval);
      this.interval = null;
    }
  }

  reset = () => {
    this.setState({ totalMilliseconds: 0 });
  }

  update = () => {
    const { totalMilliseconds } = this.state;

    this.setState({ totalMilliseconds: totalMilliseconds + 1000 });
  }

  render () {
    const { totalMilliseconds } = this.state;

    return (
      <span className={baseClass}>{convertSeconds(totalMilliseconds)}</span>
    );
  }
}

export default Timer;
